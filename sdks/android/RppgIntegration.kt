// FacePhys on-device rPPG SDK — Android integration (calling flow only).
//
// This file shows how to CALL the SDK; the SDK binary (rppg-sdk.aar),
// encrypted model assets and the native model-decryption kit are delivered separately.
//
// Requirements: Android API 24+ (arm64-v8a / x86_64).
// appKey must be fetched from your own server at runtime — never hard-code it.

import android.content.Context
import com.stepmini.rppg.RppgConfig
import com.stepmini.rppg.RppgEngineImpl
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

class RppgIntegrationExample(private val context: Context) {

    private val engine = RppgEngineImpl()

    // 1 + 2. Decrypt the bundled assets and initialize the engine.
    suspend fun start(appKeyFromYourServer: String) {
        // Decrypt the encrypted model assets (IO thread).
        val modelBytes = withContext(Dispatchers.IO) {
            ModelDecryptor.decryptAsset(context, "step_mini.tflite.enc")
        }
        val stateBytes = withContext(Dispatchers.IO) {
            ModelDecryptor.decryptAsset(context, "state_mini.bin.enc")
        }

        val result = engine.initializeAsync(
            RppgConfig(
                context    = context,
                modelBytes = modelBytes,
                stateBytes = stateBytes,
                appKey     = appKeyFromYourServer,
                sdkInitUrl = "https://www.facephys.com/endsdk/sdk/init"
            )
        )
        if (!result.success) {
            // Handle result.code / result.message (see error codes).
        }
    }

    // 3 + 4. Feed each camera frame (e.g. from CameraX ImageAnalysis) and query metrics.
    suspend fun onCameraFrame(rgb9x9: FloatArray, dtSec: Float) {
        val frameResult = engine.processFrame(rgb9x9, dtSec)

        // Query metrics once enough data is collected (~30s).
        if (frameResult.hasEnoughWindow) {
            val metrics = engine.getMetricsAsync()
            val hr  = metrics.hr    // Float? (null when data insufficient)
            val sqi = metrics.sqi   // Float? (0..1)
        }
    }

    // 5. Release when done.
    fun stop() {
        engine.release()
    }
}
