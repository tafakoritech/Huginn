package ir.tafakoritech.huginn.core.external

import android.content.Context
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.android.qualifiers.ApplicationContext
import dagger.hilt.components.SingletonComponent
import javax.inject.Singleton

@Module
@InstallIn(SingletonComponent::class)
object ExternalModule {

    @Provides
    @Singleton
    fun provideCallLauncher(@ApplicationContext context: Context): CallLauncher =
        SystemIntentCallLauncher(context)

    @Provides
    @Singleton
    fun provideSmsLauncher(@ApplicationContext context: Context): SmsLauncher =
        SystemSmsLauncher(context)

    @Provides
    @Singleton
    fun provideMessengerLauncher(@ApplicationContext context: Context): MessengerLauncher =
        SystemMessengerLauncher(context)

    @Provides
    @Singleton
    fun provideEmailLauncher(@ApplicationContext context: Context): EmailLauncher =
        SystemEmailLauncher(context)

    @Provides
    @Singleton
    fun provideContactInsertLauncher(@ApplicationContext context: Context): ContactInsertLauncher =
        SystemContactInsertLauncher(context)
}
