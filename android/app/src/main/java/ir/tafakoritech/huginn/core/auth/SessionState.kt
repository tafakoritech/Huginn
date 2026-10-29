package ir.tafakoritech.huginn.core.auth

import ir.tafakoritech.huginn.data.model.User

/** Global authentication state observed by the navigation host. */
sealed interface SessionState {
    data object Unknown : SessionState

    data class Authenticated(val user: User) : SessionState

    data object Anonymous : SessionState
}
