package ir.tafakoritech.huginn

import ir.tafakoritech.huginn.core.api.HuginnApi
import ir.tafakoritech.huginn.core.util.AppResult
import ir.tafakoritech.huginn.core.util.getOrNull
import ir.tafakoritech.huginn.data.mock.MockHuginnApi
import ir.tafakoritech.huginn.data.model.FollowUp
import ir.tafakoritech.huginn.data.model.FollowUpCreateRequest
import ir.tafakoritech.huginn.data.model.FollowUpStatus
import ir.tafakoritech.huginn.data.model.FollowUpsResponse
import ir.tafakoritech.huginn.data.repository.FollowUpRepository
import ir.tafakoritech.huginn.feature.followups.FollowUpTab
import ir.tafakoritech.huginn.feature.followups.FollowUpsUiState
import ir.tafakoritech.huginn.feature.followups.FollowUpsViewModel
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.TestScope
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config

@OptIn(ExperimentalCoroutinesApi::class)
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [34])
class FollowUpsViewModelTest {

    private fun vm(api: HuginnApi = MockHuginnApi()): FollowUpsViewModel {
        val repo = FollowUpRepository(
            api = api,
            json = Json { ignoreUnknownKeys = true; coerceInputValues = true },
        )
        return FollowUpsViewModel(repo)
    }

    private fun runVmTest(block: suspend TestScope.() -> Unit) = runTest {
        Dispatchers.setMain(StandardTestDispatcher(testScheduler))
        try {
            block()
        } finally {
            Dispatchers.resetMain()
        }
    }

    @Test
    fun `today tab loads real follow-ups`() = runVmTest {
        val viewModel = vm()
        advanceUntilIdle()
        val state = viewModel.state.value
        assertTrue("expected Loaded, got $state", state is FollowUpsUiState.Loaded)
        val loaded = state as FollowUpsUiState.Loaded
        assertEquals(FollowUpTab.TODAY, loaded.activeTab)
        assertEquals("fu-1", loaded.items.first().id)
        assertEquals("شرکت نمونه ۱", loaded.items.first().companyName)
    }

    @Test
    fun `completed tab is a real empty state`() = runVmTest {
        val viewModel = vm()
        viewModel.selectTab(FollowUpTab.COMPLETED)
        advanceUntilIdle()
        assertTrue("expected Empty, got ${viewModel.state.value}", viewModel.state.value is FollowUpsUiState.Empty)
    }

    @Test
    fun `API failure surfaces an Error state distinct from empty`() = runVmTest {
        val mock = MockHuginnApi().apply { failFollowUps = true }
        val viewModel = vm(mock)
        advanceUntilIdle()
        val state = viewModel.state.value
        assertTrue("expected Error, got $state", state is FollowUpsUiState.Error)
    }

    @Test
    fun `complete removes the item from the pending view`() = runVmTest {
        val viewModel = vm()
        advanceUntilIdle()
        viewModel.complete("fu-1")
        advanceUntilIdle()
        val state = viewModel.state.value
        assertTrue(state is FollowUpsUiState.Loaded || state is FollowUpsUiState.Empty)
        val items = (state as? FollowUpsUiState.Loaded)?.items.orEmpty()
        assertTrue("completed follow-up must leave the pending list", items.none { it.id == "fu-1" })
    }

    @Test
    fun `reschedule moves an overdue follow-up to pending`() = runVmTest {
        val viewModel = vm()
        viewModel.selectTab(FollowUpTab.OVERDUE)
        advanceUntilIdle()
        val overdue = viewModel.state.value as FollowUpsUiState.Loaded
        assertEquals("fu-3", overdue.items.first().id)

        viewModel.reschedule("fu-3", "2099-01-01T10:00:00Z")
        advanceUntilIdle()
        val state = viewModel.state.value
        assertTrue("overdue view must be empty after reschedule, got $state", state is FollowUpsUiState.Empty)
    }

    @Test
    fun `create adds a follow-up to the lead and refreshes`() = runVmTest {
        val viewModel = vm()
        advanceUntilIdle()
        viewModel.create(
            FollowUpCreateRequest(
                leadId = "lead-1",
                dueAt = java.time.Instant.now().toString(),
                title = "تماس جدید",
                description = "یادآوری",
            ),
        )
        advanceUntilIdle()
        val state = viewModel.state.value
        assertTrue("expected Loaded after create, got $state", state is FollowUpsUiState.Loaded)
        assertTrue((state as FollowUpsUiState.Loaded).items.any { it.title == "تماس جدید" })
    }

    @Test
    fun `refresh reloads the current tab`() = runVmTest {
        val viewModel = vm()
        viewModel.selectTab(FollowUpTab.UPCOMING)
        advanceUntilIdle()
        val loaded = viewModel.state.value as FollowUpsUiState.Loaded
        assertEquals("fu-2", loaded.items.first().id)

        viewModel.refresh()
        advanceUntilIdle()
        val after = viewModel.state.value
        assertTrue(after is FollowUpsUiState.Loaded)
        assertEquals("fu-2", (after as FollowUpsUiState.Loaded).items.first().id)
    }

    @Test
    fun `reopen moves a completed follow-up back to pending`() = runVmTest {
        val viewModel = vm()
        viewModel.selectTab(FollowUpTab.COMPLETED)
        advanceUntilIdle()
        assertTrue(viewModel.state.value is FollowUpsUiState.Empty)

        // Complete fu-1 on the today tab, then it appears under completed.
        viewModel.selectTab(FollowUpTab.TODAY)
        advanceUntilIdle()
        viewModel.complete("fu-1")
        advanceUntilIdle()
        viewModel.selectTab(FollowUpTab.COMPLETED)
        advanceUntilIdle()
        val completed = viewModel.state.value as? FollowUpsUiState.Loaded
        assertTrue(completed != null && completed.items.any { it.id == "fu-1" && it.status == FollowUpStatus.COMPLETED })

        viewModel.reopen("fu-1")
        advanceUntilIdle()
        val afterReopen = viewModel.state.value
        assertTrue("reopened must leave completed, got $afterReopen", afterReopen is FollowUpsUiState.Empty)
    }
}
