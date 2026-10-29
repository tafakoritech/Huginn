# Huginn ProGuard / R8 rules

# Retrofit 2
-keepattributes Signature, InnerClasses, EnclosingMethod, *Annotation*
-keepattributes RuntimeVisibleAnnotations, RuntimeVisibleParameterAnnotations
-keep class retrofit2.** { *; }
-dontwarn retrofit2.**
-dontwarn javax.annotation.**

# OkHttp 4
-dontwarn okhttp3.**
-dontwarn okio.**
-keep class okhttp3.** { *; }

# Kotlinx Serialization
-keepattributes RuntimeVisibleAnnotations, AnnotationDefault
-keep,includedescriptorclasses class ir.tafakoritech.huginn.**$$serializer { *; }
-keepclassmembers class ir.tafakoritech.huginn.** {
    *** Companion;
}
-keepclasseswithmembers class ir.tafakoritech.huginn.** {
    kotlinx.serialization.KSerializer serializer(...);
}
-if @kotlinx.serialization.Serializable class ir.tafakoritech.huginn.**
-keepclassmembers class ir.tafakoritech.huginn.** {
    kotlinx.serialization.KSerializer serializer(...);
}

# Kotlin
-dontwarn kotlinx.serialization.**
-keep class kotlinx.serialization.** { *; }

# Room
-keep class * extends androidx.room.RoomDatabase
-keep @androidx.room.Entity class *
-dontwarn androidx.room.**

# WorkManager
-keep class * extends androidx.work.Worker
-keepclassmembers class * extends androidx.work.Worker {
    public <init>(android.content.Context, androidx.work.WorkerParameters);
}
-keep class androidx.hilt.work.** { *; }

# Hilt
-keep class dagger.hilt.** { *; }
-keep class javax.inject.** { *; }
-dontwarn dagger.hilt.**

# Tink / security-crypto (missing annotations are compile-time only)
-dontwarn com.google.errorprone.annotations.**
-dontwarn com.google.crypto.tink.**

# Gson/Jackson not used; kotlinx serialization handles DTOs.
