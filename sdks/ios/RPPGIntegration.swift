// FacePhys on-device rPPG SDK — iOS integration (calling flow only).
//
// This file shows how to CALL the SDK; the SDK binary (RPPGSDK.xcframework),
// encrypted model assets and the model-decryption kit are delivered separately.
//
// Requirements: iOS 15+, Xcode 15+, Swift 5.9+.
// appKey must be fetched from your own server at runtime — never hard-code it.

import RPPGSDK

final class RPPGIntegrationExample: NSObject, RPPGDelegate {

    private let manager = RPPGManager()

    func start(appKeyFromYourServer: String) throws {
        manager.delegate = self

        // 1. Decrypt the bundled, encrypted model assets.
        let modelData = try ModelDecryptor.decryptBundleResource(named: "step_mini", withExtension: "tflite.enc")
        let stateData = try ModelDecryptor.decryptBundleResource(named: "state",     withExtension: "gz.enc")

        // 2. Initialize the engine.
        let config = RPPGConfig.production(
            appKey:     appKeyFromYourServer,
            modelBytes: modelData,
            stateBytes: stateData
        )
        manager.initialize(config: config) { result in
            if result.success {
                print("SDK ready")
            } else {
                print("init failed: \(result.error?.localizedDescription ?? "")")
            }
        }
    }

    // 3. Feed each camera frame (e.g. from AVCaptureVideoDataOutput).
    func onCameraFrame(_ pixelBuffer: CVPixelBuffer, faceRect: CGRect, timestamp: TimeInterval) {
        manager.processFrame(pixelBuffer, faceRect: faceRect, timestamp: timestamp)
    }

    // 4. Receive results on the main thread via RPPGDelegate.
    func rppgDidCalculateResult(hr: Int, sqi: Float, breathingRate: Int?) {
        // HR available ~30s after start, delivered every ~5s.
        print("HR: \(hr) BPM  SQI: \(sqi)")
    }

    func rppgDidUpdateHrv(rrBpm: Float?, sdnnMs: Float?, rmssdMs: Float?, ibiCount: Int) {
        // HRV available ~60s after start, delivered every ~10s.
        print("SDNN: \(sdnnMs ?? 0) ms  RMSSD: \(rmssdMs ?? 0) ms")
    }

    func rppgDidEncounterError(_ error: Error) {
        print("error: \(error)")
    }

    // 5. Reset between subjects / release when done.
    func stop() {
        manager.reset()
        manager.release()
    }
}
