package ir.tafakoritech.huginn.core.database

import android.content.Context
import androidx.room.Room
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.android.qualifiers.ApplicationContext
import dagger.hilt.components.SingletonComponent
import javax.inject.Singleton

@Module
@InstallIn(SingletonComponent::class)
object DatabaseModule {

    @Provides
    @Singleton
    fun provideDatabase(@ApplicationContext context: Context): HuginnDatabase =
        Room.databaseBuilder(context, HuginnDatabase::class.java, "huginn.db")
            .fallbackToDestructiveMigration()
            .build()

    @Provides
    fun provideCachedLeadDao(db: HuginnDatabase): CachedLeadDao = db.cachedLeadDao()

    @Provides
    fun provideCachedWorkDao(db: HuginnDatabase): CachedWorkDao = db.cachedWorkDao()

    @Provides
    fun provideCachedDetailDao(db: HuginnDatabase): CachedDetailDao = db.cachedDetailDao()

    @Provides
    fun providePendingReportDao(db: HuginnDatabase): PendingReportDao = db.pendingReportDao()
}
