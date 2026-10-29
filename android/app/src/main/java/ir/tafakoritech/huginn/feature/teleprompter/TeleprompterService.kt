package ir.tafakoritech.huginn.feature.teleprompter

import android.app.Service
import android.content.Context
import android.content.Intent
import android.graphics.PixelFormat
import android.os.Build
import android.os.Bundle
import android.os.IBinder
import android.view.Gravity
import android.view.MotionEvent
import android.view.View
import android.view.WindowManager
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.detectDragGestures
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Call
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.DragHandle
import androidx.compose.material.icons.filled.KeyboardArrowDown
import androidx.compose.material.icons.filled.KeyboardArrowUp
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.ComposeView
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleOwner
import androidx.lifecycle.LifecycleRegistry
import androidx.lifecycle.ViewModelStore
import androidx.lifecycle.ViewModelStoreOwner
import androidx.lifecycle.setViewTreeLifecycleOwner
import androidx.lifecycle.setViewTreeViewModelStoreOwner
import androidx.savedstate.SavedStateRegistry
import androidx.savedstate.SavedStateRegistryController
import androidx.savedstate.SavedStateRegistryOwner
import androidx.savedstate.setViewTreeSavedStateRegistryOwner
import ir.tafakoritech.huginn.core.theme.HuginnTheme

/**
 * Floating overlay service that keeps the call scenario visible as a live
 * teleprompter during a phone call.
 */
class TeleprompterService : Service() {

    private var windowManager: WindowManager? = null
    private var overlayView: View? = null
    private var layoutParams: WindowManager.LayoutParams? = null

    private var opening: String = ""
    private var intro: String = ""
    private var reason: String = ""
    private var value: String = ""
    private var questions: List<String> = emptyList()
    private var objections: List<String> = emptyList()
    private var responses: List<String> = emptyList()
    private var nextStep: String = ""
    private var closing: String = ""
    private var leadName: String? = null
    private var contactName: String? = null

    private val lifecycleOwner = object : LifecycleOwner, SavedStateRegistryOwner {
        private val lifecycleRegistry = LifecycleRegistry(this)
        private val savedStateRegistryController = SavedStateRegistryController.create(this)
        override val lifecycle: Lifecycle get() = lifecycleRegistry
        override val savedStateRegistry: SavedStateRegistry get() = savedStateRegistryController.savedStateRegistry

        fun init() {
            savedStateRegistryController.performRestore(null)
            lifecycleRegistry.handleLifecycleEvent(Lifecycle.Event.ON_CREATE)
            lifecycleRegistry.handleLifecycleEvent(Lifecycle.Event.ON_START)
            lifecycleRegistry.handleLifecycleEvent(Lifecycle.Event.ON_RESUME)
        }

        fun destroy() {
            lifecycleRegistry.handleLifecycleEvent(Lifecycle.Event.ON_PAUSE)
            lifecycleRegistry.handleLifecycleEvent(Lifecycle.Event.ON_STOP)
            lifecycleRegistry.handleLifecycleEvent(Lifecycle.Event.ON_DESTROY)
        }
    }

    private val viewModelStoreOwner = object : ViewModelStoreOwner {
        override val viewModelStore: ViewModelStore = ViewModelStore()
    }

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        if (intent == null) return START_NOT_STICKY

        opening = intent.getStringExtra(EXTRA_OPENING).orEmpty()
        intro = intent.getStringExtra(EXTRA_INTRO).orEmpty()
        reason = intent.getStringExtra(EXTRA_REASON).orEmpty()
        value = intent.getStringExtra(EXTRA_VALUE).orEmpty()
        questions = intent.getStringArrayListExtra(EXTRA_QUESTIONS) ?: emptyList()
        objections = intent.getStringArrayListExtra(EXTRA_OBJECTIONS) ?: emptyList()
        responses = intent.getStringArrayListExtra(EXTRA_RESPONSES) ?: emptyList()
        nextStep = intent.getStringExtra(EXTRA_NEXT_STEP).orEmpty()
        closing = intent.getStringExtra(EXTRA_CLOSING).orEmpty()
        leadName = intent.getStringExtra(EXTRA_LEAD_NAME)
        contactName = intent.getStringExtra(EXTRA_CONTACT_NAME)

        showOverlay()
        return START_NOT_STICKY
    }

    private fun showOverlay() {
        if (overlayView != null) return

        windowManager = getSystemService(Context.WINDOW_SERVICE) as WindowManager

        val overlayType = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            WindowManager.LayoutParams.TYPE_APPLICATION_OVERLAY
        } else {
            @Suppress("DEPRECATION")
            WindowManager.LayoutParams.TYPE_PHONE
        }

        val params = WindowManager.LayoutParams(
            WindowManager.LayoutParams.WRAP_CONTENT,
            WindowManager.LayoutParams.WRAP_CONTENT,
            overlayType,
            WindowManager.LayoutParams.FLAG_NOT_FOCUSABLE or
                WindowManager.LayoutParams.FLAG_NOT_TOUCH_MODAL or
                WindowManager.LayoutParams.FLAG_LAYOUT_IN_SCREEN,
            PixelFormat.TRANSLUCENT,
        ).apply {
            gravity = Gravity.TOP or Gravity.START
            x = 30
            y = 120
        }
        layoutParams = params

        lifecycleOwner.init()

        val composeView = ComposeView(this).apply {
            setViewTreeLifecycleOwner(lifecycleOwner)
            setViewTreeSavedStateRegistryOwner(lifecycleOwner)
            setViewTreeViewModelStoreOwner(viewModelStoreOwner)

            setContent {
                HuginnTheme(darkTheme = true) {
                    TeleprompterOverlayContent(
                        leadName = leadName ?: contactName ?: "سناریوی تماس",
                        opening = opening,
                        intro = intro,
                        reason = reason,
                        value = value,
                        questions = questions,
                        objections = objections,
                        responses = responses,
                        nextStep = nextStep,
                        closing = closing,
                        onDrag = { dx, dy ->
                            params.x = (params.x + dx.toInt()).coerceAtLeast(0)
                            params.y = (params.y + dy.toInt()).coerceAtLeast(0)
                            windowManager?.updateViewLayout(this, params)
                        },
                        onClose = {
                            stopSelf()
                        },
                    )
                }
            }
        }

        overlayView = composeView
        windowManager?.addView(composeView, params)
    }

    override fun onDestroy() {
        super.onDestroy()
        lifecycleOwner.destroy()
        viewModelStoreOwner.viewModelStore.clear()
        overlayView?.let { view ->
            try {
                windowManager?.removeView(view)
            } catch (_: Exception) {
            }
            overlayView = null
        }
    }

    companion object {
        const val EXTRA_OPENING = "extra_opening"
        const val EXTRA_INTRO = "extra_intro"
        const val EXTRA_REASON = "extra_reason"
        const val EXTRA_VALUE = "extra_value"
        const val EXTRA_QUESTIONS = "extra_questions"
        const val EXTRA_OBJECTIONS = "extra_objections"
        const val EXTRA_RESPONSES = "extra_responses"
        const val EXTRA_NEXT_STEP = "extra_next_step"
        const val EXTRA_CLOSING = "extra_closing"
        const val EXTRA_LEAD_NAME = "extra_lead_name"
        const val EXTRA_CONTACT_NAME = "extra_contact_name"
    }
}

@Composable
private fun TeleprompterOverlayContent(
    leadName: String,
    opening: String,
    intro: String,
    reason: String,
    value: String,
    questions: List<String>,
    objections: List<String>,
    responses: List<String>,
    nextStep: String,
    closing: String,
    onDrag: (Float, Float) -> Unit,
    onClose: () -> Unit,
) {
    var minimized by remember { mutableStateOf(false) }
    var fontScale by remember { mutableFloatStateOf(16f) }

    val darkBackground = Color(0xF0181E29)
    val headerBackground = Color(0xFF222D3D)
    val borderColor = Color(0x3338BDF8)
    val textColor = Color(0xFFF1F5F9)
    val labelColor = Color(0xFF38BDF8)
    val secondaryTextColor = Color(0xFFCBD5E1)

    if (minimized) {
        Surface(
            modifier = Modifier
                .padding(8.dp)
                .shadow(12.dp, RoundedCornerShape(28.dp))
                .border(1.5.dp, borderColor, RoundedCornerShape(28.dp))
                .clip(RoundedCornerShape(28.dp))
                .background(headerBackground)
                .clickable { minimized = false }
                .pointerInput(Unit) {
                    detectDragGestures { change, dragAmount ->
                        change.consume()
                        onDrag(dragAmount.x, dragAmount.y)
                    }
                },
            color = headerBackground,
            shape = RoundedCornerShape(28.dp),
        ) {
            Row(
                modifier = Modifier.padding(horizontal = 14.dp, vertical = 10.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                Icon(
                    imageVector = Icons.Default.Call,
                    contentDescription = null,
                    tint = labelColor,
                    modifier = Modifier.size(18.dp),
                )
                Text(
                    text = leadName,
                    color = textColor,
                    fontSize = 13.sp,
                    fontWeight = FontWeight.SemiBold,
                    maxLines = 1,
                )
                Icon(
                    imageVector = Icons.Default.KeyboardArrowDown,
                    contentDescription = "Expand",
                    tint = labelColor,
                    modifier = Modifier.size(18.dp),
                )
            }
        }
    } else {
        Surface(
            modifier = Modifier
                .padding(8.dp)
                .widthIn(min = 280.dp, max = 340.dp)
                .heightIn(max = 420.dp)
                .shadow(16.dp, RoundedCornerShape(16.dp))
                .border(1.dp, borderColor, RoundedCornerShape(16.dp))
                .clip(RoundedCornerShape(16.dp)),
            color = darkBackground,
            shape = RoundedCornerShape(16.dp),
        ) {
            Column(modifier = Modifier.fillMaxWidth()) {
                // Drag handle / Header bar
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .background(headerBackground)
                        .pointerInput(Unit) {
                            detectDragGestures { change, dragAmount ->
                                change.consume()
                                onDrag(dragAmount.x, dragAmount.y)
                            }
                        }
                        .padding(horizontal = 10.dp, vertical = 6.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.SpaceBetween,
                ) {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(6.dp),
                        modifier = Modifier.weight(1f),
                    ) {
                        Icon(
                            imageVector = Icons.Default.DragHandle,
                            contentDescription = "Drag",
                            tint = Color(0xFF94A3B8),
                            modifier = Modifier.size(18.dp),
                        )
                        Text(
                            text = leadName,
                            color = textColor,
                            fontSize = 13.sp,
                            fontWeight = FontWeight.Bold,
                            maxLines = 1,
                        )
                    }

                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(2.dp),
                    ) {
                        // Font size control A-
                        Box(
                            modifier = Modifier
                                .size(28.dp)
                                .clip(CircleShape)
                                .clickable {
                                    if (fontScale > 12f) fontScale -= 2f
                                },
                            contentAlignment = Alignment.Center,
                        ) {
                            Text(
                                text = "A-",
                                color = Color(0xFF94A3B8),
                                fontSize = 11.sp,
                                fontWeight = FontWeight.Bold,
                            )
                        }

                        // Font size control A+
                        Box(
                            modifier = Modifier
                                .size(28.dp)
                                .clip(CircleShape)
                                .clickable {
                                    if (fontScale < 24f) fontScale += 2f
                                },
                            contentAlignment = Alignment.Center,
                        ) {
                            Text(
                                text = "A+",
                                color = labelColor,
                                fontSize = 12.sp,
                                fontWeight = FontWeight.Bold,
                            )
                        }

                        // Minimize button
                        Box(
                            modifier = Modifier
                                .size(28.dp)
                                .clip(CircleShape)
                                .clickable { minimized = true },
                            contentAlignment = Alignment.Center,
                        ) {
                            Icon(
                                imageVector = Icons.Default.KeyboardArrowUp,
                                contentDescription = "Minimize",
                                tint = Color(0xFF94A3B8),
                                modifier = Modifier.size(18.dp),
                            )
                        }

                        // Close button
                        Box(
                            modifier = Modifier
                                .size(28.dp)
                                .clip(CircleShape)
                                .clickable(onClick = onClose),
                            contentAlignment = Alignment.Center,
                        ) {
                            Icon(
                                imageVector = Icons.Default.Close,
                                contentDescription = "Close",
                                tint = Color(0xFFEF4444),
                                modifier = Modifier.size(18.dp),
                            )
                        }
                    }
                }

                // Scrollable teleprompter scenario content
                Column(
                    modifier = Modifier
                        .fillMaxWidth()
                        .weight(1f, fill = false)
                        .verticalScroll(rememberScrollState())
                        .padding(horizontal = 12.dp, vertical = 10.dp),
                    verticalArrangement = Arrangement.spacedBy(10.dp),
                ) {
                    if (opening.isNotBlank()) {
                        TeleprompterSection("جمله شروع", opening, fontScale, labelColor, textColor)
                    }
                    if (intro.isNotBlank()) {
                        TeleprompterSection("معرفی", intro, fontScale, labelColor, textColor)
                    }
                    if (reason.isNotBlank()) {
                        TeleprompterSection("دلیل تماس", reason, fontScale, labelColor, textColor)
                    }
                    if (value.isNotBlank()) {
                        TeleprompterSection("ارزش پیشنهادی", value, fontScale, labelColor, textColor)
                    }
                    if (questions.isNotEmpty()) {
                        Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                            Text(
                                text = "سؤالات کشف نیاز:",
                                color = labelColor,
                                fontSize = (fontScale - 3f).coerceAtLeast(11f).sp,
                                fontWeight = FontWeight.SemiBold,
                            )
                            questions.forEachIndexed { idx, q ->
                                Text(
                                    text = "${idx + 1}. $q",
                                    color = textColor,
                                    fontSize = fontScale.sp,
                                    lineHeight = (fontScale * 1.35f).sp,
                                )
                            }
                        }
                    }
                    if (objections.isNotEmpty()) {
                        Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                            Text(
                                text = "اعتراض‌ها و پاسخ‌ها:",
                                color = labelColor,
                                fontSize = (fontScale - 3f).coerceAtLeast(11f).sp,
                                fontWeight = FontWeight.SemiBold,
                            )
                            objections.forEachIndexed { idx, o ->
                                Text(
                                    text = "اعتراض: $o",
                                    color = Color(0xFFFCA5A5),
                                    fontSize = fontScale.sp,
                                    lineHeight = (fontScale * 1.35f).sp,
                                )
                                val resp = responses.getOrNull(idx)
                                if (!resp.isNullOrBlank()) {
                                    Text(
                                        text = "پاسخ: $resp",
                                        color = Color(0xFF86EFAC),
                                        fontSize = fontScale.sp,
                                        lineHeight = (fontScale * 1.35f).sp,
                                        modifier = Modifier.padding(start = 8.dp),
                                    )
                                }
                            }
                        }
                    }
                    if (nextStep.isNotBlank()) {
                        TeleprompterSection("قدم بعدی", nextStep, fontScale, labelColor, textColor)
                    }
                    if (closing.isNotBlank()) {
                        TeleprompterSection("جمله پایانی", closing, fontScale, labelColor, textColor)
                    }
                }
            }
        }
    }
}

@Composable
private fun TeleprompterSection(
    label: String,
    text: String,
    fontSize: Float,
    labelColor: Color,
    textColor: Color,
) {
    Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
        Text(
            text = "$label:",
            color = labelColor,
            fontSize = (fontSize - 3f).coerceAtLeast(11f).sp,
            fontWeight = FontWeight.SemiBold,
        )
        Text(
            text = text,
            color = textColor,
            fontSize = fontSize.sp,
            lineHeight = (fontSize * 1.35f).sp,
        )
    }
}
