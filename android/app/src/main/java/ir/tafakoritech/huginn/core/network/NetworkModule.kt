package ir.tafakoritech.huginn.core.network

import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.components.SingletonComponent
import ir.tafakoritech.huginn.BuildConfig
import ir.tafakoritech.huginn.core.api.HuginnApi
import ir.tafakoritech.huginn.core.api.LlmHuginnApi
import ir.tafakoritech.huginn.core.api.RefreshApi
import ir.tafakoritech.huginn.core.auth.AuthInterceptor
import ir.tafakoritech.huginn.core.auth.RefreshInterceptor
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import retrofit2.Retrofit
import retrofit2.converter.kotlinx.serialization.asConverterFactory
import java.util.concurrent.TimeUnit
import javax.inject.Named
import javax.inject.Singleton

@Module
@InstallIn(SingletonComponent::class)
object NetworkModule {

    const val BASE_URL = "huginn_base_url"
    const val AUTH_CLIENT = "huginn_auth_client"
    const val REFRESH_CLIENT = "huginn_refresh_client"
    const val LLM_CLIENT = "huginn_llm_client"

    @Provides
    @Singleton
    fun provideJson(): Json = Json {
        ignoreUnknownKeys = true
        explicitNulls = false
        encodeDefaults = true
        isLenient = true
        coerceInputValues = true
    }

    @Provides
    @Named(BASE_URL)
    fun provideBaseUrl(): String {
        val configured = BuildConfig.HUGINN_API_BASE_URL
        val normalized = if (configured.endsWith("/")) configured else "$configured/"
        require(normalized.startsWith("https://") || normalized.startsWith("http://")) {
            "HUGINN_API_BASE_URL must start with http(s)://"
        }
        return normalized
    }

    @Provides
    @Singleton
    fun provideLoggingInterceptor(): RequestLoggingInterceptor = RequestLoggingInterceptor()

    @Provides
    @Singleton
    @Named(REFRESH_CLIENT)
    fun provideRefreshClient(
        logging: RequestLoggingInterceptor,
    ): OkHttpClient =
        OkHttpClient.Builder()
            .connectTimeout(15, TimeUnit.SECONDS)
            .readTimeout(20, TimeUnit.SECONDS)
            .writeTimeout(20, TimeUnit.SECONDS)
            .addInterceptor(logging)
            .build()

    @Provides
    @Singleton
    @Named(AUTH_CLIENT)
    fun provideAuthClient(
        authInterceptor: AuthInterceptor,
        refreshInterceptor: RefreshInterceptor,
        logging: RequestLoggingInterceptor,
    ): OkHttpClient =
        OkHttpClient.Builder()
            .connectTimeout(15, TimeUnit.SECONDS)
            .readTimeout(30, TimeUnit.SECONDS)
            .writeTimeout(30, TimeUnit.SECONDS)
            .addInterceptor(authInterceptor)
            .addInterceptor(refreshInterceptor)
            .addInterceptor(logging)
            .build()

    /**
     * Dedicated long-running client for LLM-backed endpoints (call-scenario
     * generation). The backend's LLM per-attempt timeout is 180s and the whole
     * generation operation is bounded at ~185s, so this client must tolerate a
     * response that takes minutes to arrive. Deliberate outer layer: it always
     * lives slightly LONGER than the backend operation so the backend never
     * sees a spurious client-side cancellation mid-generation.
     *
     * Only endpoints that genuinely wait on the LLM use this client; every
     * other request keeps the tight 30s timeouts of the shared auth client.
     */
    @Provides
    @Singleton
    @Named(LLM_CLIENT)
    fun provideLlmClient(
        authInterceptor: AuthInterceptor,
        refreshInterceptor: RefreshInterceptor,
        logging: RequestLoggingInterceptor,
    ): OkHttpClient =
        OkHttpClient.Builder()
            .connectTimeout(30, TimeUnit.SECONDS)
            .writeTimeout(60, TimeUnit.SECONDS)
            .readTimeout(200, TimeUnit.SECONDS)
            .callTimeout(250, TimeUnit.SECONDS)
            .addInterceptor(authInterceptor)
            .addInterceptor(refreshInterceptor)
            .addInterceptor(logging)
            .build()

    @Provides
    @Singleton
    fun provideRetrofit(
        @Named(AUTH_CLIENT) client: OkHttpClient,
        @Named(BASE_URL) baseUrl: String,
        json: Json,
    ): Retrofit =
        Retrofit.Builder()
            .baseUrl(baseUrl)
            .client(client)
            .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
            .build()

    @Provides
    @Singleton
    fun provideRefreshApi(
        @Named(REFRESH_CLIENT) client: OkHttpClient,
        @Named(BASE_URL) baseUrl: String,
        json: Json,
    ): RefreshApi =
        Retrofit.Builder()
            .baseUrl(baseUrl)
            .client(client)
            .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
            .build()
            .create(RefreshApi::class.java)

    @Provides
    @Singleton
    fun provideHuginnApi(
        retrofit: Retrofit,
    ): HuginnApi = retrofit.create(HuginnApi::class.java)

    @Provides
    @Singleton
    fun provideLlmHuginnApi(
        @Named(LLM_CLIENT) client: OkHttpClient,
        @Named(BASE_URL) baseUrl: String,
        json: Json,
    ): LlmHuginnApi =
        Retrofit.Builder()
            .baseUrl(baseUrl)
            .client(client)
            .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
            .build()
            .create(LlmHuginnApi::class.java)
}
