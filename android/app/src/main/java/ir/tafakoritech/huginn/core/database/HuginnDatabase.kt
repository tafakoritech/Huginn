package ir.tafakoritech.huginn.core.database

import androidx.room.ColumnInfo
import androidx.room.Dao
import androidx.room.Database
import androidx.room.Delete
import androidx.room.Entity
import androidx.room.Insert
import androidx.room.OnConflictStrategy
import androidx.room.PrimaryKey
import androidx.room.Query

/** Cached lead list items for offline browsing. */
@Entity(tableName = "cached_leads")
data class CachedLeadEntity(
    @PrimaryKey val id: String,
    val json: String,
    @ColumnInfo(name = "saved_at") val savedAt: Long = System.currentTimeMillis(),
)

/** Cached work queue snapshot for offline home. */
@Entity(tableName = "cached_work_queue")
data class CachedWorkItemEntity(
    @PrimaryKey val leadId: String,
    val json: String,
    @ColumnInfo(name = "saved_at") val savedAt: Long = System.currentTimeMillis(),
)

/** Cached recently viewed lead details. */
@Entity(tableName = "cached_lead_details")
data class CachedLeadDetailEntity(
    @PrimaryKey val id: String,
    val json: String,
    @ColumnInfo(name = "saved_at") val savedAt: Long = System.currentTimeMillis(),
)

/**
 * Locally persisted mandatory report awaiting backend confirmation.
 * Kept pending until the server acknowledges submission (ProcessDeath safe).
 */
@Entity(tableName = "pending_report_queue")
data class PendingReportLocalEntity(
    @PrimaryKey @ColumnInfo(name = "interaction_id") val interactionId: String,
    @ColumnInfo(name = "lead_id") val leadId: String,
    @ColumnInfo(name = "report_json") val reportJson: String,
    @ColumnInfo(name = "attempt_count") val attemptCount: Int = 0,
    @ColumnInfo(name = "created_at") val createdAt: Long = System.currentTimeMillis(),
    @ColumnInfo(name = "last_attempt_at") val lastAttemptAt: Long? = null,
)

@Dao
interface CachedLeadDao {
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsert(entity: CachedLeadEntity)

    @Query("DELETE FROM cached_leads")
    suspend fun clearAll()

    @Query("SELECT * FROM cached_leads ORDER BY saved_at DESC LIMIT :limit")
    suspend fun getAll(limit: Int = 50): List<CachedLeadEntity>
}

@Dao
interface CachedWorkDao {
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsert(entity: CachedWorkItemEntity)

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsertAll(entities: List<CachedWorkItemEntity>)

    @Query("DELETE FROM cached_work_queue")
    suspend fun clearAll()

    @Query("SELECT * FROM cached_work_queue ORDER BY saved_at DESC")
    suspend fun getAll(): List<CachedWorkItemEntity>
}

@Dao
interface CachedDetailDao {
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsert(entity: CachedLeadDetailEntity)

    @Query("SELECT * FROM cached_lead_details WHERE id = :id")
    suspend fun getById(id: String): CachedLeadDetailEntity?

    @Query("DELETE FROM cached_lead_details")
    suspend fun clearAll()
}

@Dao
interface PendingReportDao {
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsert(entity: PendingReportLocalEntity)

    @Query("SELECT * FROM pending_report_queue ORDER BY created_at ASC")
    suspend fun getAll(): List<PendingReportLocalEntity>

    @Query("SELECT * FROM pending_report_queue WHERE interaction_id = :id LIMIT 1")
    suspend fun getById(id: String): PendingReportLocalEntity?

    @Query("UPDATE pending_report_queue SET attempt_count = attempt_count + 1, last_attempt_at = :now WHERE interaction_id = :id")
    suspend fun incrementAttempt(id: String, now: Long)

    @Query("DELETE FROM pending_report_queue WHERE interaction_id = :id")
    suspend fun delete(id: String)

    @Delete
    suspend fun delete(entity: PendingReportLocalEntity)

    @Query("SELECT count(*) FROM pending_report_queue")
    suspend fun count(): Int
}

@Database(
    entities = [
        CachedLeadEntity::class,
        CachedWorkItemEntity::class,
        CachedLeadDetailEntity::class,
        PendingReportLocalEntity::class,
    ],
    version = 1,
    exportSchema = true,
)
abstract class HuginnDatabase : androidx.room.RoomDatabase() {
    abstract fun cachedLeadDao(): CachedLeadDao
    abstract fun cachedWorkDao(): CachedWorkDao
    abstract fun cachedDetailDao(): CachedDetailDao
    abstract fun pendingReportDao(): PendingReportDao
}
