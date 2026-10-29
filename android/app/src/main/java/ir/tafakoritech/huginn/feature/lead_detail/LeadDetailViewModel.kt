package ir.tafakoritech.huginn.feature.lead_detail

import androidx.lifecycle.SavedStateHandle
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import android.content.Context
import dagger.hilt.android.lifecycle.HiltViewModel
import dagger.hilt.android.qualifiers.ApplicationContext
import ir.tafakoritech.huginn.core.external.CallLauncher
import ir.tafakoritech.huginn.core.external.ContactInsertLauncher
import ir.tafakoritech.huginn.core.external.EmailLauncher
import ir.tafakoritech.huginn.core.external.LaunchResult
import ir.tafakoritech.huginn.core.external.MessengerChannels
import ir.tafakoritech.huginn.core.external.MessengerLauncher
import ir.tafakoritech.huginn.core.external.SmsLauncher
import ir.tafakoritech.huginn.core.util.AppResult
import ir.tafakoritech.huginn.core.util.getOrNull
import ir.tafakoritech.huginn.data.model.CallScenario
import ir.tafakoritech.huginn.data.model.FollowUp
import ir.tafakoritech.huginn.data.model.Contact
import ir.tafakoritech.huginn.data.model.DeepLink
import ir.tafakoritech.huginn.data.model.InteractionChannel
import ir.tafakoritech.huginn.data.model.InteractionObjective
import ir.tafakoritech.huginn.data.model.LeadDetail
import ir.tafakoritech.huginn.data.model.Note
import ir.tafakoritech.huginn.data.model.TimelineEvent
import ir.tafakoritech.huginn.data.repository.FollowUpRepository
import ir.tafakoritech.huginn.data.repository.InteractionRepository
import ir.tafakoritech.huginn.data.repository.LeadRepository
import ir.tafakoritech.huginn.data.repository.QualificationRepository
import ir.tafakoritech.huginn.feature.messaging.ChannelAvailability
import ir.tafakoritech.huginn.feature.messaging.ChannelAvailabilityEvaluator
import ir.tafakoritech.huginn.feature.teleprompter.TeleprompterManager
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

data class MessagingDraft(
    val interactionId: String,
    val channel: String,
    val deepLink: DeepLink?,
    val text: String,
    val subject: String?,
)

/** Pre-call assistance and teleprompter UI state. */
data class CallScenarioUi(
    val visible: Boolean = false,
    val loading: Boolean = false,
    val error: Boolean = false,
    val saving: Boolean = false,
    val saveError: Boolean = false,
    val startInEditMode: Boolean = false,
    val scenario: CallScenario? = null,
)

data class LeadDetailUiState(
    val loading: Boolean = true,
    val offline: Boolean = false,
    val error: Boolean = false,
    val lead: LeadDetail? = null,
    val selectedContactId: String? = null,
    val callInProgress: Boolean = false,
    val callPromptVisible: Boolean = false,
    val scenario: CallScenarioUi = CallScenarioUi(),
    val scenarios: List<CallScenario> = emptyList(),
    val loadingScenarios: Boolean = false,
    val overlayPermissionDialogVisible: Boolean = false,
    val pendingCallScenario: CallScenario? = null,
    val deleteScenarioTarget: CallScenario? = null,
    val messageSheetVisible: Boolean = false,
    val suggestedSheet: MessagingDraft? = null,
    val loadingSuggested: Boolean = false,
    val suggestError: Boolean = false,
    val contactSaveChannel: String? = null,
    val savingContact: Boolean = false,
    val notice: String? = null,
    val timeline: List<TimelineEvent> = emptyList(),
    val notes: List<Note> = emptyList(),
    val followUps: List<FollowUp> = emptyList(),
    val followUpsLoading: Boolean = false,
)

@HiltViewModel
class LeadDetailViewModel @Inject constructor(
    savedStateHandle: SavedStateHandle,
    private val leadRepository: LeadRepository,
    private val interactionRepository: InteractionRepository,
    private val followUpRepository: FollowUpRepository,
    private val qualificationRepository: QualificationRepository,
    private val callLauncher: CallLauncher,
    private val smsLauncher: SmsLauncher,
    private val messengerLauncher: MessengerLauncher,
    private val emailLauncher: EmailLauncher,
    private val contactInsertLauncher: ContactInsertLauncher,
    @ApplicationContext private val appContext: Context,
) : ViewModel() {

    private val leadId: String = checkNotNull(savedStateHandle["leadId"])

    private val _state = MutableStateFlow(LeadDetailUiState())
    val state: StateFlow<LeadDetailUiState> = _state

    private var lastInteractionId: String? = null

    /**
     * Guards stale suggested-message responses: incremented on every generation
     * and on dismissal so a late response can never reopen a dismissed composer.
     */
    private var suggestionGeneration = 0

    /** Guards stale call-scenario responses (same rationale). */
    private var scenarioGeneration = 0

    init {
        reload()
    }

    fun reload() {
        viewModelScope.launch {
            _state.value = _state.value.copy(loading = _state.value.lead == null)
            when (val result = leadRepository.leadDetail(leadId)) {
                is AppResult.Success -> {
                    val lead = result.data
                    _state.value = _state.value.copy(
                        loading = false,
                        offline = false,
                        error = false,
                        lead = lead,
                        selectedContactId = _state.value.selectedContactId
                            ?: lead.contacts.firstOrNull { it.isPrimaryContact }?.id
                            ?: lead.contacts.firstOrNull()?.id,
                    )
                    loadTimeline()
                    loadNotes()
                    loadScenarios()
                    loadLeadFollowUps()
                }
                is AppResult.Failure -> {
                    _state.value = _state.value.copy(
                        loading = false,
                        error = true,
                        offline = _state.value.lead == null,
                    )
                }
            }
        }
    }

    private fun loadTimeline() {
        viewModelScope.launch {
            val items = leadRepository.timeline(leadId).getOrNull()?.items.orEmpty()
            _state.value = _state.value.copy(timeline = items)
        }
    }

    private fun loadNotes() {
        viewModelScope.launch {
            val items = leadRepository.notes(leadId).getOrNull()?.items.orEmpty()
            _state.value = _state.value.copy(notes = items)
        }
    }

    fun loadScenarios() {
        viewModelScope.launch {
            _state.value = _state.value.copy(loadingScenarios = true)
            when (val result = interactionRepository.getCallScenarios(leadId)) {
                is AppResult.Success -> {
                    _state.value = _state.value.copy(
                        scenarios = result.data,
                        loadingScenarios = false,
                    )
                }
                is AppResult.Failure -> {
                    _state.value = _state.value.copy(loadingScenarios = false)
                }
            }
        }
    }

    fun selectContact(contact: Contact) {
        _state.value = _state.value.copy(selectedContactId = contact.id)
    }

    fun addNote(text: String) {
        if (text.isBlank()) return
        viewModelScope.launch {
            when (val result = leadRepository.addNote(leadId, text)) {
                is AppResult.Success -> {
                    _state.value = _state.value.copy(notes = listOf(result.data) + _state.value.notes)
                }
                is AppResult.Failure -> {
                    _state.value = _state.value.copy(notice = result.error::class.simpleName)
                }
            }
        }
    }

    fun consumeNotice() {
        _state.value = _state.value.copy(notice = null)
    }

    // ---- Call & Scenarios ----

    /**
     * Call button handler. Instead of launching the dialer immediately, we ask
     * whether the employee wants a suggested call scenario first. "No" continues
     * straight to the normal call flow; "Yes" opens the scenario screen and only
     * after the scenario is shown can the employee start the real call.
     */
    fun startCall() {
        val contact = selectedContact() ?: return
        if (_state.value.lead == null) return
        if (_state.value.callInProgress) return
        if (contact.primaryNumber.isNullOrBlank()) {
            _state.value = _state.value.copy(notice = "no_number")
            return
        }
        _state.value = _state.value.copy(callPromptVisible = true)
    }

    /** "No, call now": dismiss the prompt and launch the normal call flow. */
    fun proceedWithoutScenario() {
        if (_state.value.callInProgress) return
        _state.value = _state.value.copy(callPromptVisible = false)
        doCall()
    }

    fun dismissCallPrompt() {
        _state.value = _state.value.copy(callPromptVisible = false)
    }

    /** "Yes, generate scenario": open the scenario screen with a loading state. */
    fun requestCallScenario() {
        val ui = _state.value.scenario
        if (ui.loading) return // never duplicate a generation request
        _state.value = _state.value.copy(
            callPromptVisible = false,
            scenario = CallScenarioUi(visible = true, loading = true),
        )
        generateCallScenario()
    }

    /** Retry generating the call scenario after a failure. */
    fun retryCallScenario() {
        if (_state.value.scenario.loading) return
        _state.value = _state.value.copy(scenario = CallScenarioUi(visible = true, loading = true))
        generateCallScenario()
    }

    private fun generateCallScenario() {
        val lead = _state.value.lead ?: return
        val contact = selectedContact()
        val generation = ++scenarioGeneration
        viewModelScope.launch {
            // The backend generates the scenario from the lead's real context and saves it;
            // the client only sends the lead/contact identifiers.
            val result = interactionRepository.callScenario(lead.id, contact)
            if (generation != scenarioGeneration) return@launch // dismissed meanwhile
            when (result) {
                is AppResult.Success -> {
                    _state.value = _state.value.copy(
                        scenario = CallScenarioUi(
                            visible = true,
                            scenario = result.data.scenario,
                        ),
                    )
                    loadScenarios()
                }
                is AppResult.Failure -> {
                    // Keep the scenario screen open so the user can retry or cancel.
                    _state.value = _state.value.copy(
                        scenario = CallScenarioUi(visible = true, error = true),
                    )
                }
            }
        }
    }

    /** Opens an existing or manual scenario in the sheet. */
    fun openScenario(scenario: CallScenario, editMode: Boolean = false) {
        _state.value = _state.value.copy(
            scenario = CallScenarioUi(
                visible = true,
                scenario = scenario,
                startInEditMode = editMode,
            ),
        )
    }

    /** Prepares a blank scenario for creation in edit mode. */
    fun createNewScenario() {
        _state.value = _state.value.copy(
            scenario = CallScenarioUi(
                visible = true,
                scenario = CallScenario(leadId = leadId),
                startInEditMode = true,
            ),
        )
    }

    /** Persists edited scenario text to the backend database. */
    fun saveCallScenario(scenario: CallScenario) {
        val lead = _state.value.lead ?: return
        val currentUi = _state.value.scenario
        if (currentUi.saving) return
        _state.value = _state.value.copy(
            scenario = currentUi.copy(saving = true, saveError = false),
        )
        viewModelScope.launch {
            val result = if (!scenario.id.isNullOrBlank()) {
                interactionRepository.updateCallScenario(lead.id, scenario)
            } else {
                interactionRepository.createCallScenario(lead.id, scenario.copy(leadId = lead.id))
            }
            when (result) {
                is AppResult.Success -> {
                    val saved = result.data
                    _state.value = _state.value.copy(
                        scenario = CallScenarioUi(
                            visible = true,
                            scenario = saved,
                            saving = false,
                            saveError = false,
                            startInEditMode = false,
                        ),
                        notice = "scenario_saved",
                    )
                    loadScenarios()
                }
                is AppResult.Failure -> {
                    _state.value = _state.value.copy(
                        scenario = _state.value.scenario.copy(
                            saving = false,
                            saveError = true,
                        ),
                        notice = "scenario_save_failed",
                    )
                }
            }
        }
    }

    /** Requests confirmation to delete an archived scenario. */
    fun deleteCallScenario(scenario: CallScenario) {
        _state.value = _state.value.copy(deleteScenarioTarget = scenario)
    }

    fun dismissDeleteScenario() {
        _state.value = _state.value.copy(deleteScenarioTarget = null)
    }

    fun confirmDeleteScenario() {
        val target = _state.value.deleteScenarioTarget ?: return
        val targetId = target.id ?: return
        _state.value = _state.value.copy(deleteScenarioTarget = null)
        viewModelScope.launch {
            when (val result = interactionRepository.deleteCallScenario(leadId, targetId)) {
                is AppResult.Success -> {
                    _state.value = _state.value.copy(notice = "scenario_deleted")
                    loadScenarios()
                }
                is AppResult.Failure -> {
                    _state.value = _state.value.copy(notice = "scenario_delete_failed")
                }
            }
        }
    }

    /** "Start Call" with scenario: launches call and shows live teleprompter. */
    fun startCallWithScenario(scenario: CallScenario? = null) {
        if (_state.value.callInProgress) return
        val targetScenario = scenario ?: _state.value.scenario.scenario ?: return
        scenarioGeneration++
        _state.value = _state.value.copy(scenario = CallScenarioUi())

        if (TeleprompterManager.canDrawOverlays(appContext)) {
            val lead = _state.value.lead
            val contact = selectedContact()
            TeleprompterManager.start(
                context = appContext,
                scenario = targetScenario,
                leadName = lead?.companyName,
                contactName = contact?.displayName,
            )
            doCall()
        } else {
            // Permission needed for teleprompter overlay
            _state.value = _state.value.copy(
                overlayPermissionDialogVisible = true,
                pendingCallScenario = targetScenario,
            )
        }
    }

    /** Opens only the live teleprompter overlay for this scenario. */
    fun openTeleprompter(scenario: CallScenario? = null) {
        val targetScenario = scenario ?: _state.value.scenario.scenario ?: return
        if (TeleprompterManager.canDrawOverlays(appContext)) {
            _state.value = _state.value.copy(scenario = CallScenarioUi())
            val lead = _state.value.lead
            val contact = selectedContact()
            TeleprompterManager.start(
                context = appContext,
                scenario = targetScenario,
                leadName = lead?.companyName,
                contactName = contact?.displayName,
            )
        } else {
            _state.value = _state.value.copy(
                overlayPermissionDialogVisible = true,
                pendingCallScenario = targetScenario,
            )
        }
    }

    fun proceedCallWithoutTeleprompter() {
        _state.value = _state.value.copy(
            overlayPermissionDialogVisible = false,
            pendingCallScenario = null,
        )
        doCall()
    }

    fun dismissOverlayPermissionDialog() {
        _state.value = _state.value.copy(
            overlayPermissionDialogVisible = false,
            pendingCallScenario = null,
        )
    }

    fun dismissCallScenario() {
        scenarioGeneration++
        _state.value = _state.value.copy(scenario = CallScenarioUi())
    }

    /** The actual phone-call flow: record the interaction then open the dialer. */
    private fun doCall() {
        val contact = selectedContact() ?: return
        val lead = _state.value.lead ?: return
        if (_state.value.callInProgress) return
        if (contact.primaryNumber.isNullOrBlank()) {
            _state.value = _state.value.copy(notice = "no_number")
            return
        }
        viewModelScope.launch {
            _state.value = _state.value.copy(callInProgress = true)
            when (val action = interactionRepository.startAction(lead.id, contact, InteractionChannel.CALL, InteractionObjective.FIRST_CONTACT)) {
                is AppResult.Success -> {
                    lastInteractionId = action.data.interactionId
                    val phone = action.data.phone ?: contact.primaryNumber ?: ""
                    // Opens the system dialer; the mandatory report is triggered by
                    // the backend's pending-reports check on resume.
                    when (val result = callLauncher.launch(phone)) {
                        LaunchResult.Launched -> {}
                        else -> _state.value = _state.value.copy(notice = "call_failed")
                    }
                }
                is AppResult.Failure -> {
                    _state.value = _state.value.copy(notice = resultErrorKey(action.error))
                }
            }
            _state.value = _state.value.copy(callInProgress = false)
        }
    }

    // ---- Messaging ----

    fun openMessageSheet() {
        _state.value = _state.value.copy(messageSheetVisible = true)
    }

    fun dismissMessageSheet() {
        _state.value = _state.value.copy(messageSheetVisible = false)
    }

    fun onSelectChannel(channel: String) {
        val contact = selectedContact() ?: return
        val lead = _state.value.lead ?: return
        _state.value = _state.value.copy(messageSheetVisible = false)
        val status = ChannelAvailabilityEvaluator.evaluate(
            channel = channel,
            contact = contact,
            permissions = lead.permissions,
            messengerInstalled = messengerInstalled(channel),
        )
        when (status.availability) {
            ChannelAvailability.ENABLED -> openComposerAndGenerate(channel, contact)
            // Rubika / Eitaa are save-contact only.
            ChannelAvailability.CONTACT_SAVE_REQUIRED -> {
                _state.value = _state.value.copy(contactSaveChannel = channel)
            }
            ChannelAvailability.DISABLED_NOT_INSTALLED -> {
                _state.value = _state.value.copy(notice = "not_installed")
            }
            ChannelAvailability.DISABLED_NO_TARGET -> {
                _state.value = _state.value.copy(notice = "no_target")
            }
            ChannelAvailability.DISABLED_NO_NUMBER -> {
                _state.value = _state.value.copy(notice = "no_number")
            }
            ChannelAvailability.DISABLED_NO_EMAIL -> {
                _state.value = _state.value.copy(notice = "no_email")
            }
            else -> {
                _state.value = _state.value.copy(notice = "channel_unavailable")
            }
        }
    }

    /**
     * Open the composer immediately (with a loading placeholder) so the user
     * gets instant feedback, then fetch the backend-suggested message. The
     * composer stays open while generating and on failure (with a retry), so a
     * single channel tap is always enough.
     */
    private fun openComposerAndGenerate(channel: String, contact: Contact) {
        if (_state.value.loadingSuggested) return // never duplicate a request
        _state.value = _state.value.copy(
            messageSheetVisible = false,
            suggestedSheet = MessagingDraft(
                interactionId = "",
                channel = channel,
                deepLink = null,
                text = "",
                subject = null,
            ),
            suggestError = false,
        )
        generateSuggested(channel, contact)
    }

    /** Retry generating the suggested message after a failure. */
    fun retrySuggested() {
        if (_state.value.loadingSuggested) return
        val draft = _state.value.suggestedSheet ?: return
        val contact = selectedContact() ?: return
        _state.value = _state.value.copy(suggestError = false)
        generateSuggested(draft.channel, contact)
    }

    /** Keep the draft text in sync with user edits in the composer. */
    fun updateSuggestedText(text: String) {
        val draft = _state.value.suggestedSheet ?: return
        _state.value = _state.value.copy(suggestedSheet = draft.copy(text = text))
    }

    private fun generateSuggested(channel: String, contact: Contact) {
        if (_state.value.loadingSuggested) return // prevent duplicate backend calls
        val lead = _state.value.lead ?: return
        val generation = ++suggestionGeneration
        viewModelScope.launch {
            _state.value = _state.value.copy(loadingSuggested = true)
            // The backend generates the suggested message + deep link in the
            // contact-actions response; it is never composed client-side.
            val action = interactionRepository.startAction(lead.id, contact, channel, InteractionObjective.FIRST_CONTACT)
            if (generation != suggestionGeneration) return@launch // dismissed meanwhile
            when (action) {
                is AppResult.Success -> {
                    val a = action.data
                    _state.value = _state.value.copy(
                        loadingSuggested = false,
                        suggestedSheet = MessagingDraft(
                            interactionId = a.interactionId,
                            channel = channel,
                            deepLink = a.deepLink,
                            text = a.suggestedMessage.orEmpty(),
                            subject = a.suggestedSubject,
                        ),
                    )
                }
                is AppResult.Failure -> {
                    // Keep the composer open so the user can retry or cancel.
                    _state.value = _state.value.copy(
                        loadingSuggested = false,
                        suggestError = true,
                    )
                }
            }
        }
    }

    fun openMessenger(draft: MessagingDraft, text: String) {
        val contact = selectedContact() ?: return
        viewModelScope.launch {
            val result = when (draft.channel) {
                InteractionChannel.SMS -> smsLauncher.launch(contact.primaryNumber.orEmpty(), text)
                InteractionChannel.EMAIL -> emailLauncher.launch(contact.email, draft.subject, text)
                // Phone is passed so Telegram/Bale can build phone-based links.
                else -> messengerLauncher.launch(draft.channel, draft.deepLink, contact.primaryNumber, text)
            }
            when (result) {
                LaunchResult.Launched -> _state.value = _state.value.copy(suggestedSheet = null, loadingSuggested = false, suggestError = false)
                // Only reachable for save-contact-only channels (Rubika/Eitaa).
                LaunchResult.ContactSaveRequired -> {
                    _state.value = _state.value.copy(suggestedSheet = null, loadingSuggested = false, suggestError = false, contactSaveChannel = draft.channel)
                }
                // Direct chat impossible: keep the draft so the message is not lost.
                LaunchResult.NotInstalled -> {
                    _state.value = _state.value.copy(notice = "not_installed")
                }
            }
        }
    }

    fun dismissSuggestedSheet() {
        suggestionGeneration++
        _state.value = _state.value.copy(suggestedSheet = null, loadingSuggested = false, suggestError = false)
    }

    fun saveContactFor(channel: String) {
        val contact = selectedContact() ?: return
        val lead = _state.value.lead ?: return
        viewModelScope.launch {
            _state.value = _state.value.copy(savingContact = true)
            val result = contactInsertLauncher.launch(
                name = contact.displayName.ifBlank { lead.companyName },
                phone = contact.primaryNumber,
                company = lead.companyName,
                jobTitle = contact.jobTitle,
                email = contact.email,
            )
            _state.value = _state.value.copy(savingContact = false, contactSaveChannel = null)
            // Save-contact-only channels (Rubika/Eitaa): never attempt a direct
            // chat afterwards — the employee finds the person inside the app.
            _state.value = _state.value.copy(
                notice = if (result == LaunchResult.Launched) "contact_saved" else "channel_unavailable",
            )
        }
    }

    fun dismissContactSave() {
        _state.value = _state.value.copy(contactSaveChannel = null)
    }

    // ---- Follow-ups ----

    private fun loadLeadFollowUps() {
        viewModelScope.launch {
            _state.value = _state.value.copy(followUpsLoading = true)
            val items = followUpRepository.forLead(leadId).getOrNull().orEmpty()
            _state.value = _state.value.copy(followUps = items, followUpsLoading = false)
        }
    }

    fun completeFollowUp(id: String) {
        viewModelScope.launch {
            followUpRepository.complete(id)
            loadLeadFollowUps()
        }
    }

    fun createFollowUp(dueAtIso: String, description: String?, title: String? = null) {
        viewModelScope.launch {
            when (val result = followUpRepository.create(
                ir.tafakoritech.huginn.data.model.FollowUpCreateRequest(
                    leadId = leadId,
                    dueAt = dueAtIso,
                    title = title,
                    description = description,
                ),
            )) {
                is AppResult.Success -> loadLeadFollowUps()
                is AppResult.Failure -> _state.value = _state.value.copy(notice = resultErrorKey(result.error))
            }
        }
    }

    fun rescheduleFollowUp(id: String, dueAtIso: String) {
        viewModelScope.launch {
            followUpRepository.reschedule(id, dueAtIso)
            loadLeadFollowUps()
        }
    }

    fun cancelFollowUp(id: String) {
        viewModelScope.launch {
            followUpRepository.cancel(id)
            loadLeadFollowUps()
        }
    }

    fun reopenFollowUp(id: String) {
        viewModelScope.launch {
            followUpRepository.reopen(id)
            loadLeadFollowUps()
        }
    }

    fun qualify(inputs: Map<String, Boolean>) {
        viewModelScope.launch {
            when (val result = qualificationRepository.qualify(leadId, inputs, null)) {
                is AppResult.Success -> reload()
                is AppResult.Failure -> _state.value = _state.value.copy(notice = resultErrorKey(result.error))
            }
        }
    }

    fun disqualify(reason: String, explanation: String?) {
        viewModelScope.launch {
            when (val result = qualificationRepository.disqualify(leadId, reason, explanation)) {
                is AppResult.Success -> reload()
                is AppResult.Failure -> _state.value = _state.value.copy(notice = resultErrorKey(result.error))
            }
        }
    }

    fun selectedContact(): Contact? {
        val lead = _state.value.lead ?: return null
        return lead.contacts.firstOrNull { it.id == _state.value.selectedContactId }
            ?: lead.contacts.firstOrNull { it.isPrimaryContact }
            ?: lead.contacts.firstOrNull()
    }

    fun messengerInstalled(channel: String): Boolean =
        MessengerChannels.isInstalled(appContext, channel)

    private fun resultErrorKey(error: ir.tafakoritech.huginn.core.util.AppError): String = when (error) {
        ir.tafakoritech.huginn.core.util.AppError.Conflict -> "conflict"
        ir.tafakoritech.huginn.core.util.AppError.Forbidden -> "forbidden"
        ir.tafakoritech.huginn.core.util.AppError.Network -> "network"
        else -> "error"
    }
}
