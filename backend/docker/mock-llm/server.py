"""Development LLM mock for the Huginn ingestion pipeline.

Serves OpenAI-compatible /chat/completions on :9991 and returns a valid
structured extraction payload so the full pipeline (upload -> extract ->
analyze -> leads -> dedup -> batch COMPLETED) can run end-to-end without an
external LLM or internet access.

NOT for production.
"""
import json
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

PORT = 9991

CANDIDATES = {
    "candidates": [
        {
            "company_name": "شرکت نوآوران بسته‌بندی تیسا",
            "display_name": "نوآوران بسته‌بندی",
            "business_type": "تولیدکننده",
            "industry": "بسته‌بندی",
            "description": "تولید انواع بسته‌بندی صنعتی و کارتن",
            "province": "تهران",
            "city": "تهران",
            "address": "شهرک صنعتی چهاردانگه",
            "website": None,
            "instagram": None,
            "contacts": [
                {"full_name": "امیر رضایی", "job_title": "مدیر فروش", "mobile": "09121112233",
                 "phone": "02155667788", "email": None, "is_decision_maker": True}
            ],
            "evidence": {"field_name": "company_name", "source_excerpt": "تولید انواع بسته‌بندی صنعتی و کارتن",
                         "source_row": 2, "confidence": 0.96},
        },
        {
            "company_name": "کارخانه نساجی سبلان آریا",
            "display_name": None,
            "business_type": "تولیدکننده",
            "industry": "نساجی",
            "description": "تولید پارچه و منسوجات",
            "province": "اردبیل",
            "city": "اردبیل",
            "address": "شهرک صنعتی",
            "website": None,
            "instagram": None,
            "contacts": [
                {"full_name": "نگار کریمی", "job_title": "مسئول خرید", "mobile": "09195556677",
                 "phone": None, "email": "info@ariatex.ir", "is_decision_maker": False}
            ],
            "evidence": {"field_name": "company_name", "source_excerpt": "تولید پارچه و منسوجات",
                         "source_row": 8, "confidence": 0.93},
        },
        {
            "company_name": "صنایع چوبی درنا",
            "display_name": None,
            "business_type": "تولیدکننده",
            "industry": "مبلمان",
            "description": "تولید مبلمان اداری و خانگی",
            "province": "خراسان رضوی",
            "city": "مشهد",
            "address": "جاده سنتو",
            "website": "www.dorna-wood.ir",
            "instagram": None,
            "contacts": [
                {"full_name": "مدیرعامل", "job_title": "مدیرعامل", "mobile": None,
                 "phone": "05138445566", "email": None, "is_decision_maker": True}
            ],
            "evidence": {"field_name": "company_name", "source_excerpt": "تولید مبلمان اداری و خانگی",
                         "source_row": 13, "confidence": 0.9},
        },
    ]
}


class Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        length = int(self.headers.get("Content-Length", 0))
        body_text = b""
        if length:
            body_text = self.rfile.read(length)
        request = {}
        try:
            request = json.loads(body_text.decode("utf-8"))
        except Exception:
            pass

        prompt_blob = ""
        for m in request.get("messages", []):
            prompt_blob += (m.get("content") or "") + "\n"

        if "scenario" in prompt_blob.lower() or "sales_focus" in prompt_blob.lower():
            content = json.dumps({
                "strategy": {
                    "lead_summary": "the specific prospect from the RECIPIENT block",
                    "fit": "medium",
                    "fit_summary": "project value maps to a plausible pain for this lead",
                    "primary_angle": "lead-specific value angle chosen from the analysis",
                    "secondary_angle": "discovery-led angle",
                    "likely_pain": "hypothesis to be validated via discovery questions",
                    "uncertainties": ["the lead's current workflow"],
                    "call_objective": "understand workflow and schedule a short follow-up",
                    "tone": "natural, concise spoken Persian"
                },
                "opening_sentence": "سلام، وقت شما بخیر. متوجه شدم مجموعه شما در حال توسعه است؛ درباره این تماس صحبت کنیم؟",
                "introduction": "اینجا نماینده فروش هستم؛ بابت پروژه‌ای که برای کسب‌وکارهای فعال در این حوزه طراحی شده تماس می‌گیرم.",
                "reason_for_call": "در چارچوب پروژه جاری، فکر کردیم همکاری با مجموعه شما می‌تواند ارزش واقعی ایجاد کند.",
                "value_proposition": "پروژه ما به طور مشخص روی کاهش هزینه عملیاتی و رشد فروش شما اثر می‌گذارد.",
                "discovery_questions": [
                    "در حال حاضر بزرگ‌ترین چالش کسب‌وکار شما چیست؟",
                    "الان چطور این فرآیند را مدیریت می‌کنید؟",
                    "اگر راهی برای بهبود این وضعیت باشد، چه چیزی برای شما مهم‌تر است؟"
                ],
                "discovery_question_purposes": [
                    {"question": "در حال حاضر بزرگ‌ترین چالش کسب‌وکار شما چیست؟", "purpose": "surface the lead's most pressing pain"},
                    {"question": "الان چطور این فرآیند را مدیریت می‌کنید؟", "purpose": "understand the current workflow and cost"},
                    {"question": "اگر راهی برای بهبود این وضعیت باشد، چه چیزی برای شما مهم‌تر است؟", "purpose": "identify buying criteria and motivation"}
                ],
                "likely_objections": [
                    "الان وقت ندارم.",
                    "بودجه کافی نداریم.",
                    "با تأمین‌کننده دیگری کار می‌کنیم."
                ],
                "objection_responses": [
                    "حتماً، فقط چند دقیقه وقت می‌خواهم.",
                    "متوجه‌ام؛ شاید بتوانیم بعداً درباره ارزش بلندمدت صحبت کنیم.",
                    "مشکلی نیست؛ اجازه می‌دهید اطلاعات کوتاهی بفرستم تا مقایسه کنید؟"
                ],
                "conversation_branches": [
                    {"condition": "Lead already outsources the work", "response_strategy": "ask who provides it and what they pay"},
                    {"condition": "Lead does everything in-house", "response_strategy": "probe time spent and quality ceiling"}
                ],
                "recommended_next_step": "پیشنهاد می‌کنم یک تماس کوتاه ۱۰ دقیقه‌ای هماهنگ کنیم.",
                "closing_line": "ممنون از وقتی که گذاشتید. روز خوبی داشته باشید.",
                "context_used": {
                    "project": "project from context",
                    "lead": "lead from context",
                    "sales_angle": "angle from context"
                }
            }, ensure_ascii=False)
        else:
            content = json.dumps(CANDIDATES, ensure_ascii=False)

        body = json.dumps({
            "choices": [{"message": {"content": content}}],
            "usage": {"prompt_tokens": 100, "completion_tokens": 50},
        }, ensure_ascii=False).encode("utf-8")
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        self.send_response(200)
        self.send_header("Content-Length", "2")
        self.end_headers()
        self.wfile.write(b"ok")

    def log_message(self, *args):
        pass


if __name__ == "__main__":
    ThreadingHTTPServer(("0.0.0.0", PORT), Handler).serve_forever()
