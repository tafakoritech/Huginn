package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/tafakorpardazan/huginn/internal/config"
)

// Job types published on RabbitMQ.
const (
	JobDocumentExtract = "document.extract"
	JobBatchAnalyze    = "batch.analyze"
	JobLeadExtract     = "lead.extract"
	JobLeadDedupe      = "lead.dedupe"
	JobLeadScore       = "lead.score"
	JobMessageGenerate = "message.generate"
	JobMuninnExport    = "muninn.export"
)

// Job is the message envelope placed on the queue.
type Job struct {
	Type      string         `json:"type"`
	ID        string         `json:"id"`
	Payload   map[string]any `json:"payload"`
	Retry     int            `json:"retry"`
	CreatedAt time.Time      `json:"created_at"`
}

const (
	exchangeName    = "huginn"
	queuePrefix     = "huginn."
	deadLetterQueue = "huginn.dead"

	defaultRetryLimit = 5
	defaultRetryDelay = 5 * time.Second
)

// Client publishes and consumes jobs on RabbitMQ.
type Client struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	channels []*amqp.Channel // consumer channels, closed on Client.Close

	logger         *slog.Logger
	retryLimit     int
	retryBaseDelay time.Duration
}

// Handler processes a job; return nil to ack. Errors are classified by the
// queue: permanent errors dead-letter immediately, everything else retries
// with backoff up to the configured limit, then dead-letters.
type Handler func(ctx context.Context, job Job) error

// publisher abstracts the AMQP publish primitive so the delivery loop is
// unit-testable without a live broker.
type publisher interface {
	PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
}

func New(ctx context.Context, cfg config.RabbitMQConfig) (*Client, error) {
	conn, err := amqp.DialConfig(cfg.URL, amqp.Config{
		Heartbeat: 10 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("rabbitmq dial: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("rabbitmq channel: %w", err)
	}
	if err := ch.Qos(10, 0, false); err != nil {
		return nil, fmt.Errorf("rabbitmq qos: %w", err)
	}

	c := &Client{
		conn:           conn,
		channel:        ch,
		logger:         slog.Default(),
		retryLimit:     cfg.JobRetryLimit,
		retryBaseDelay: cfg.JobRetryDelay,
	}
	if c.retryLimit <= 0 {
		c.retryLimit = defaultRetryLimit
	}
	if c.retryBaseDelay <= 0 {
		c.retryBaseDelay = defaultRetryDelay
	}
	if err := c.setup(); err != nil {
		conn.Close()
		return nil, err
	}
	return c, nil
}

// SetLogger attaches a structured logger used for job lifecycle observability.
func (c *Client) SetLogger(l *slog.Logger) {
	if l != nil {
		c.logger = l
	}
}

func (c *Client) setup() error {
	if err := c.channel.ExchangeDeclare(exchangeName, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("rabbitmq declare exchange: %w", err)
	}
	// Dead letter exchange + queue
	if err := c.channel.ExchangeDeclare(exchangeName+".dlx", "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("rabbitmq declare dlx: %w", err)
	}
	dlqArgs := amqp.Table{}
	_, err := c.channel.QueueDeclare(deadLetterQueue, true, false, false, false, dlqArgs)
	if err != nil {
		return fmt.Errorf("rabbitmq declare dead queue: %w", err)
	}
	if err := c.channel.QueueBind(deadLetterQueue, deadLetterQueue, exchangeName+".dlx", false, nil); err != nil {
		return err
	}
	return nil
}

// declareJobQueue creates the queue for a job type with DLX and retry config.
// Any message nacked with requeue=false is routed via the DLX to huginn.dead.
func (c *Client) declareJobQueue(jobType string) (string, error) {
	qname := queuePrefix + jobType
	args := amqp.Table{
		"x-dead-letter-exchange":    exchangeName + ".dlx",
		"x-dead-letter-routing-key": deadLetterQueue,
	}
	if _, err := c.channel.QueueDeclare(qname, true, false, false, false, args); err != nil {
		return "", err
	}
	if err := c.channel.QueueBind(qname, jobType, exchangeName, false, nil); err != nil {
		return "", err
	}
	return qname, nil
}

// Publish sends a job. The routing key equals the job type.
func (c *Client) Publish(ctx context.Context, job Job) error {
	if _, err := c.declareJobQueue(job.Type); err != nil {
		return err
	}
	body, err := json.Marshal(job)
	if err != nil {
		return err
	}
	err = c.channel.PublishWithContext(ctx, exchangeName, job.Type, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
		Timestamp:    time.Now(),
	})
	if err != nil {
		return fmt.Errorf("rabbitmq publish %s: %w", job.Type, err)
	}
	return nil
}

// Consume registers a handler for a job type. Documents are processed by a
// small pool of consumers per job type so one slow document never blocks the
// rest of the queue (e.g. long LLM analysis holding up later uploads). Each
// consumer gets its OWN channel — RabbitMQ forbids multiple consumers on a
// single channel.
func (c *Client) Consume(ctx context.Context, jobType string, handler Handler) error {
	qname, err := c.declareJobQueue(jobType)
	if err != nil {
		return err
	}

	workers := 4
	if jobType == JobMuninnExport {
		workers = 1 // export ordering matters; keep it serial
	}
	for i := 0; i < workers; i++ {
		ch, err := c.conn.Channel()
		if err != nil {
			return fmt.Errorf("rabbitmq channel %s: %w", jobType, err)
		}
		c.channels = append(c.channels, ch)
		deliveries, err := ch.Consume(qname, "", false, false, false, false, nil)
		if err != nil {
			return fmt.Errorf("rabbitmq consume %s: %w", jobType, err)
		}
		go c.consumeLoop(ctx, ch, deliveries, handler)
	}
	return nil
}

func (c *Client) consumeLoop(ctx context.Context, ch *amqp.Channel, deliveries <-chan amqp.Delivery, handler Handler) {
	for {
		select {
		case <-ctx.Done():
			return
		case d, ok := <-deliveries:
			if !ok {
				return
			}
			c.handleDelivery(ctx, ch, d, handler)
		}
	}
}

// handleDelivery applies the ACK/NACK/retry/DLQ policy to a single delivery.
func (c *Client) handleDelivery(ctx context.Context, pub publisher, d amqp.Delivery, handler Handler) {
	var job Job
	if err := json.Unmarshal(d.Body, &job); err != nil {
		c.logger.Error("job malformed",
			"error_code", CodeInvalidPayload,
			"retryable", false,
			"action", "dead_letter",
			"error", err.Error(),
		)
		_ = d.Nack(false, false) // DLX -> huginn.dead
		return
	}

	processCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	err := handler(processCtx, job)
	cancel()

	if err == nil {
		_ = d.Ack(false)
		return
	}

	// Permanent failure: never requeue. Dead-letter immediately.
	if IsPermanent(err) {
		c.logger.Error("job permanently failed",
			"job_type", job.Type,
			"job_id", job.ID,
			"error_code", PermanentCode(err),
			"attempt", job.Retry,
			"retryable", false,
			"action", "dead_letter",
			"error", err.Error(),
		)
		_ = d.Nack(false, false) // DLX -> huginn.dead
		return
	}

	// Transient failure: apply exponential backoff, then dead-letter once the
	// retry limit is exhausted. A message is never requeued indefinitely.
	attempt := job.Retry + 1
	if attempt > c.retryLimit {
		c.logger.Error("job retry limit exceeded",
			"job_type", job.Type,
			"job_id", job.ID,
			"attempt", attempt,
			"max_attempts", c.retryLimit,
			"retryable", false,
			"action", "dead_letter",
			"error", err.Error(),
		)
		_ = d.Nack(false, false) // DLX -> huginn.dead
		return
	}

	delay := c.retryDelay(attempt)
	c.logger.Warn("job transient failure",
		"job_type", job.Type,
		"job_id", job.ID,
		"attempt", attempt,
		"max_attempts", c.retryLimit,
		"retryable", true,
		"action", "retry",
		"delay_ms", delay.Milliseconds(),
		"error", err.Error(),
	)

	job.Retry = attempt
	body, mErr := json.Marshal(job)
	if mErr != nil {
		c.logger.Error("job serialization failed", "job_id", job.ID, "action", "dead_letter")
		_ = d.Nack(false, false)
		return
	}
	// Republish with a delayed body then ACK the original. If the republish
	// fails, requeue the original so the job is not lost.
	go func(job Job, d amqp.Delivery, body []byte) {
		time.Sleep(delay)
		if err := pub.PublishWithContext(context.Background(), exchangeName, job.Type, false, false, amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
			Timestamp:    time.Now(),
		}); err == nil {
			_ = d.Ack(false)
		} else {
			_ = d.Nack(false, true)
		}
	}(job, d, body)
}

// retryDelay computes the exponential backoff for a 1-based attempt number.
func (c *Client) retryDelay(attempt int) time.Duration {
	mult := 1 << (attempt - 1)
	d := c.retryBaseDelay * time.Duration(mult)
	if d > time.Minute {
		return time.Minute // cap backoff at 1 minute
	}
	return d
}

func (c *Client) Close() error {
	for _, ch := range c.channels {
		_ = ch.Close()
	}
	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
