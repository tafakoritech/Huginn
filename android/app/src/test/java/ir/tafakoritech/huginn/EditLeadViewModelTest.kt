package ir.tafakoritech.huginn

import androidx.lifecycle.SavedStateHandle
import ir.tafakoritech.huginn.data.mock.MockHuginnApi
import ir.tafakoritech.huginn.data.model.LoginRequest
import ir.tafakoritech.huginn.data.repository.LeadRepository
import ir.tafakoritech.huginn.feature.lead_detail.EditLeadViewModel
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
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/** Edit-Lead ViewModel flow: load -> save -> reload, validation and authz. */
@OptIn(ExperimentalCoroutinesApi::class)
class EditLeadViewModelTest {

    private fun json(): Json = Json { ignoreUnknownKeys = true; coerceInputValues = true }

    private fun api(login: String = "agent"): MockHuginnApi =
        MockHuginnApi().also { kotlinx.coroutines.runBlocking { it.login(LoginRequest(login, "password")) } }

    private fun runVmTest(block: suspend TestScope.() -> Unit) = runTest {
        Dispatchers.setMain(StandardTestDispatcher(testScheduler))
        try {
            block()
        } finally {
            Dispatchers.resetMain()
        }
    }

    private fun vm(api: MockHuginnApi): EditLeadViewModel =
        EditLeadViewModel(SavedStateHandle(mapOf("leadId" to "lead-1")), LeadRepository(api, json()))

    @Test
    fun `load populates the form with the assignee name and edit permission`() = runVmTest {
        val vm = vm(api())
        advanceUntilIdle()
        assertEquals("عروسک‌سازی پارس", vm.state.value.companyName)
        assertEquals("علی محمدی", vm.state.value.lead?.ownerName)
        assertTrue(vm.state.value.lead?.permissions?.canEdit == true)
    }

    @Test
    fun `save persists through the backend and reload shows updated values`() = runVmTest {
        val vm = vm(api())
        advanceUntilIdle()
        vm.setCompanyName("عروسک‌سازی پارس نوین")
        vm.setBusinessType("عمده‌فروشی")
        vm.save()
        advanceUntilIdle()
        assertTrue(vm.state.value.saved)
        assertEquals(null, vm.state.value.error)

        vm.consumeSaved()
        vm.load()
        advanceUntilIdle()
        assertEquals("عروسک‌سازی پارس نوین", vm.state.value.companyName)
        assertEquals("عمده‌فروشی", vm.state.value.businessType)
    }

    @Test
    fun `save with blank company name is rejected locally`() = runVmTest {
        val vm = vm(api())
        advanceUntilIdle()
        vm.setCompanyName("   ")
        vm.save()
        assertTrue(vm.state.value.companyRequired)
        assertFalse(vm.state.value.saved)
    }

    @Test
    fun `assign updates the assignee and reload shows the new owner name`() = runVmTest {
        val vm = vm(api("manager"))
        advanceUntilIdle()
        vm.assign("u-agent-2")
        advanceUntilIdle()
        assertEquals("u-agent-2", vm.state.value.assigneeId)
        assertEquals("مریم رضایی", vm.state.value.lead?.ownerName)
    }

    @Test
    fun `viewer is not offered edit permission by backend rules`() = runVmTest {
        val vm = vm(api("viewer"))
        advanceUntilIdle()
        assertFalse(vm.state.value.lead?.permissions?.canEdit == true)
        assertNull(vm.state.value.error)
    }
}
