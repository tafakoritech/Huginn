package ir.tafakoritech.huginn.core.auth

/** Token persistence abstraction (secure implementation = TokenStorage). */
interface TokenStore {
    var accessToken: String?
    var refreshToken: String?
    var userJson: String?
    val hasTokens: Boolean
    fun saveTokens(access: String, refresh: String)
    fun clear()
}
