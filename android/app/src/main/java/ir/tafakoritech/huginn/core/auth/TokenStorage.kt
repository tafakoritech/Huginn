package ir.tafakoritech.huginn.core.auth

import android.content.Context
import android.content.SharedPreferences
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey
import dagger.hilt.android.qualifiers.ApplicationContext
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Token-safe storage using Android Keystore backed encryption
 * (EncryptedSharedPreferences, AES256-GCM values / AES256-SIV keys).
 *
 * Tokens are never logged. The file is excluded from backups via
 * android:allowBackup="false" in the manifest.
 */
@Singleton
class TokenStorage @Inject constructor(
    @ApplicationContext context: Context,
) : TokenStore {
    private val prefs: SharedPreferences by lazy {
        val masterKey = MasterKey.Builder(context)
            .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
            .build()
        EncryptedSharedPreferences.create(
            context,
            FILE_NAME,
            masterKey,
            EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
            EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM,
        )
    }

    override var accessToken: String?
        get() = prefs.getString(KEY_ACCESS_TOKEN, null)
        set(value) = prefs.edit().apply {
            if (value == null) remove(KEY_ACCESS_TOKEN) else putString(KEY_ACCESS_TOKEN, value)
        }.apply()

    override var refreshToken: String?
        get() = prefs.getString(KEY_REFRESH_TOKEN, null)
        set(value) = prefs.edit().apply {
            if (value == null) remove(KEY_REFRESH_TOKEN) else putString(KEY_REFRESH_TOKEN, value)
        }.apply()

    /** Cached authenticated user (serialized JSON) for instant session restore. */
    override var userJson: String?
        get() = prefs.getString(KEY_USER_JSON, null)
        set(value) = prefs.edit().apply {
            if (value == null) remove(KEY_USER_JSON) else putString(KEY_USER_JSON, value)
        }.apply()

    override val hasTokens: Boolean
        get() = accessToken != null && refreshToken != null

    override fun saveTokens(access: String, refresh: String) {
        prefs.edit()
            .putString(KEY_ACCESS_TOKEN, access)
            .putString(KEY_REFRESH_TOKEN, refresh)
            .apply()
    }

    override fun clear() {
        prefs.edit()
            .remove(KEY_ACCESS_TOKEN)
            .remove(KEY_REFRESH_TOKEN)
            .remove(KEY_USER_JSON)
            .apply()
    }

    private companion object {
        const val FILE_NAME = "huginn_secure_prefs"
        const val KEY_ACCESS_TOKEN = "access_token"
        const val KEY_REFRESH_TOKEN = "refresh_token"
        const val KEY_USER_JSON = "user_json"
    }
}
