package ir.tafakoritech.huginn

import ir.tafakoritech.huginn.core.api.RefreshApi
import ir.tafakoritech.huginn.core.auth.SessionManager
import ir.tafakoritech.huginn.core.auth.SessionState
import ir.tafakoritech.huginn.core.auth.TokenStore
import ir.tafakoritech.huginn.core.network.ApiException
import ir.tafakoritech.huginn.data.model.Tokens
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.launch
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class FakeTokenStore : TokenStore {
    override var accessToken: String? = "access.1"
    override var refreshToken: String? = "refresh.1"
    override var userJson: String? = null
    override val hasTokens: Boolean get() = accessToken != null && refreshToken != null
    override fun saveTokens(access: String, refresh: String) {
        accessToken = access
        refreshToken = refresh
    }
    override fun clear() {
        accessToken = null
        refreshToken = null
        userJson = null
    }
}

class FakeRefreshApi : RefreshApi {
    var callCount = 0
    var failWith401 = false
    var failWithNetwork = false

    override suspend fun refresh(refreshToken: String): Tokens {
        callCount++
        if (failWith401) throw ApiException(401, "UNAUTHORIZED", "expired refresh token")
        if (failWithNetwork) throw RuntimeException("network")
        return Tokens(accessToken = "access.2", refreshToken = "refresh.2", expiresIn = 3600)
    }
}

class SessionManagerTest {

    private val json = Json { ignoreUnknownKeys = true }

    @Test
    fun `successful refresh updates stored tokens`() = runTest {
        val store = FakeTokenStore()
        val api = FakeRefreshApi()
        val manager = SessionManager(store, api, json)

        assertTrue(manager.refreshTokens())

        assertEquals("access.2", store.accessToken)
        assertEquals("refresh.2", store.refreshToken)
        assertEquals(1, api.callCount)
    }

    @Test
    fun `concurrent refresh calls are single flight`() = runTest {
        val store = FakeTokenStore()
        val api = FakeRefreshApi()
        val manager = SessionManager(store, api, json)

        coroutineScope {
            repeat(5) { launch { manager.refreshTokens() } }
        }

        // Only one network refresh performed for the whole burst.
        assertEquals(1, api.callCount)
        assertEquals("access.2", store.accessToken)
    }

    @Test
    fun `refresh with revoked token clears session`() = runTest {
        val store = FakeTokenStore()
        val api = FakeRefreshApi().apply { failWith401 = true }
        val manager = SessionManager(store, api, json)
        manager.restoreFromStorage()

        assertFalse(manager.refreshTokens())

        assertEquals(null, store.accessToken)
        assertEquals(null, store.refreshToken)
        assertEquals(SessionState.Anonymous, manager.state.value)
    }

    @Test
    fun `network failure keeps tokens and returns false`() = runTest {
        val store = FakeTokenStore()
        val api = FakeRefreshApi().apply { failWithNetwork = true }
        val manager = SessionManager(store, api, json)

        assertFalse(manager.refreshTokens())

        // Offline is not treated as an expired session.
        assertEquals("access.1", store.accessToken)
    }

    @Test
    fun `restore from storage yields authenticated state`() = runTest {
        val store = FakeTokenStore().apply {
            userJson = """{"id":"u1","name":"علی","username":"agent","role":"sales_agent"}"""
        }
        val manager = SessionManager(store, FakeRefreshApi(), json)
        manager.restoreFromStorage()
        val state = manager.state.value
        assertTrue(state is SessionState.Authenticated)
        assertEquals("علی", (state as SessionState.Authenticated).user.name)
    }

    @Test
    fun `restore without tokens yields anonymous`() = runTest {
        val store = FakeTokenStore().apply {
            accessToken = null
            refreshToken = null
        }
        val manager = SessionManager(store, FakeRefreshApi(), json)
        manager.restoreFromStorage()
        assertEquals(SessionState.Anonymous, manager.state.value)
    }
}
