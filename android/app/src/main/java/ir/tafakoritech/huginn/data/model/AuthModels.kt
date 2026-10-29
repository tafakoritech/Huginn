package ir.tafakoritech.huginn.data.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonObject

@Serializable
data class LoginRequest(
    /** Backend field is `login` (username or email), not `username`. */
    val login: String,
    val password: String,
    val device: String? = null,
)

@Serializable
data class Tokens(
    @SerialName("access_token") val accessToken: String,
    @SerialName("refresh_token") val refreshToken: String,
    @SerialName("expires_in") val expiresIn: Long = 0,
    @SerialName("token_type") val tokenType: String = "Bearer",
)

@Serializable
data class User(
    val id: String,
    val name: String,
    val username: String,
    val email: String? = null,
    val mobile: String? = null,
    val role: String = "",
    @SerialName("is_active") val isActive: Boolean = true,
)

/** Login response: token pair plus the authenticated user. */
@Serializable
data class AuthResponse(
    @SerialName("access_token") val accessToken: String,
    @SerialName("refresh_token") val refreshToken: String,
    @SerialName("expires_in") val expiresIn: Long = 0,
    @SerialName("token_type") val tokenType: String = "Bearer",
    val user: User? = null,
)

@Serializable
data class ErrorBody(
    val code: String,
    val message: String,
    val details: JsonObject? = null,
)

@Serializable
data class ErrorEnvelope(val error: ErrorBody)
