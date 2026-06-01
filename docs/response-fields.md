# FacePhys V3 Response Field Reference

[English](response-fields.md) | [简体中文](response-fields.zh-CN.md)

This document lists common modules and fields returned by `/api/v3/video/process`. Actual fields depend on the API key `fieldSet`; modules may be absent when they are not enabled, signal quality is insufficient, or calculation conditions are not met.

Accuracy notes are reference values and are affected by video quality, lighting, head motion, face area, skin tone, and capture duration. Blood pressure, SpO2, psychological, emotion, behavior, and liveness outputs are for trend and state reference only, not a substitute for professional medical devices or clinical judgment.

## Module Index

- [`cardiac` Core Physiology](#cardiac)
- [`cardiac.hrv` Heart Rate Variability](#hrv)
- [`bp` Non-contact Blood Pressure](#bp)
- [`spo2` Non-contact SpO₂](#spo2)
- [`psych` Psychological Composite Scores](#psych)
- [`emotion` Emotion Analysis](#emotion)
- [`face_au` Facial Action Units](#face_au)
- [`behavior` Behavioral Intent Indicators](#behavior)
- [`appearance` Facial Attributes](#appearance)
- [`liveness` Liveness Detection](#liveness)
- [`billing` Billing Fields](#billing)

<a id="cardiac"></a>

## `cardiac` Core Physiology

**Pack:** Core Physiology Pack · 20,000 Token/req

Heart rate, signal quality, and per-second HR sequence. cardiac.sqi determines the reliability of all physiological indicators and should be checked first. Reference accuracy: ±2 / ±5 / ±8 bpm (resting / talking / exercise, at SQI ≥ 0.65).

| Field Path | Accuracy | Description |
| --- | --- | --- |
| `cardiac.hr` | ±2/±5/±8 bpm | Average heart rate (bpm); normal resting 60–100; < 40 extreme bradycardia; 40–60 low (athletes); 100–120 mild tachycardia; > 120 marked tachycardia — re-verify |
| `cardiac.sqi` | — | Signal quality 0–1; ≥ 0.65 high quality (psych reaches full range); 0.30–0.65 usable; 0.15–0.30 reference only (HRV error higher); < 0.15 discard. Common causes of low SQI: backlighting, dark skin (V/VI), head motion, face mask |
| `cardiac.hr_list[]` | ±2/±5/±8 bpm | Per-second HR sequence [{hr, ts}], ts in seconds. Rising trend = increasing anxiety; falling trend = calming; adjacent-frame delta > 20 bpm = signal noise |

<a id="hrv"></a>

## `cardiac.hrv` Heart Rate Variability

**Pack:** HRV Neural Pack · 30,000 Token/req

Complete HRV metrics (time-domain + frequency-domain + Poincaré scatter). Reflects autonomic nervous system state — the most information-rich dimension in contactless physiological computing. Key time-domain accuracy: SDNN / RMSSD ±5 / ±15 / ±20 ms; ibi_mean ±50 / ±100 / ±150 ms (resting / talking / exercise); frequency-domain and geometric metrics ~±10%; breathing rate ±2–5 breaths/min.

| Field Path | Accuracy | Description |
| --- | --- | --- |
| `hrv.ibi_mean` | ±50/±100/±150 ms | Mean NN interval (ms); normal 600–1000; low = faster HR; high = slower HR; reference accuracy: ±50 / ±100 / ±150 ms (resting / talking / exercise) |
| `hrv.sdnn` | ±5/±15/±20 ms | IBI standard deviation (ms); normal 50–150; low = limited autonomic regulation (stress/fatigue); high = flexible regulation. Clinical ref (Task Force 1996): > 50ms low risk; 20–50ms moderate; < 20ms high risk; reference accuracy: ±5 / ±15 / ±20 ms (resting / talking / exercise) |
| `hrv.rmssd` | ±5/±15/±20 ms | Root mean square of successive differences (ms); normal 20–80; low = low parasympathetic activity (stress/overwork); high = active parasympathetic (relaxation); reference accuracy: ±5 / ±15 / ±20 ms (resting / talking / exercise) |
| `hrv.pnn50` | ±10% | Percentage of intervals differing > 50ms; normal 5–45%; low = low vagal tone; high = high vagal tone, relaxed |
| `hrv.pnn20` | ±10% | Percentage of intervals differing > 20ms; normal 15–70%; more sensitive for short video segments |
| `hrv.LF` | ±10% | Low-frequency power (ms²); normal 200–2000; 0.04–0.15 Hz; mixed sympathetic + parasympathetic control |
| `hrv.HF` | ±10% | High-frequency power (ms²); normal 200–2000; 0.15–0.40 Hz; pure parasympathetic/vagal activity |
| `hrv.LF/HF` | ±10% | Sympatho-vagal balance ratio; normal waking 1.5–4.0; < 1.0 deep relaxation; 1.0–2.0 relaxed; 4.0–8.0 notable stress; > 8.0 high sympathetic activation |
| `hrv.TP` | ±10% | Total power (ms²); normal 500–5000; total autonomic energy reserve |
| `hrv.VLF` | ±10% | Very low frequency power (ms²); 0.003–0.04 Hz; related to thermoregulation/metabolic control; normal 50–500; < 50 very weak metabolic regulation; > 500 in short videos is likely estimation error (< 5 min recordings have low accuracy — trend reference only) |
| `hrv.sd1` | ±10% | Poincaré short axis ≈ RMSSD/√2 (ms); normal 15–45; < 10 very low parasympathetic activity (high stress/overwork); 10–20 low (mild stress); 20–45 normal; > 45 active parasympathetic (deep relaxation/recovery) |
| `hrv.sd2` | ±10% | Poincaré long axis ≈ √(2·SDNN²−½·SD1²) (ms); normal 50–150; < 40 very low long-term HRV (narrow autonomic regulation range); 40–80 low; 80–150 normal; > 150 high (well-trained athletes) |
| `hrv.sd1_sd2` | ±10% | SD1/SD2 ratio; normal 0.3–0.7; > 0.7 strong parasympathetic dominance (deep relaxation); 0.5–0.7 relaxed; 0.3–0.5 autonomic balance; < 0.3 sympathetic dominance / chronic stress |
| `hrv.poincare_s` | ±10% | Poincaré scatter area = π×SD1×SD2 (ms²); < 2000 very low autonomic regulation space (high risk); 2000–5000 low; 5000–15000 normal; > 15000 high (athletes/high-HRV individuals) |
| `hrv.breathing_rate` | ±2–5 breaths/min | Breathing rate (breaths/min); extracted from rPPG signal, accuracy ±2–5 breaths/min. Ranges: < 8 intentional slowing/deep meditation; 8–12 deep relaxation; 12–16 normal waking rest; 16–20 mildly active/talking; 20–25 mild stress/post-exercise; > 25 hyperventilation (anxiety/panic). At 6 breaths/min (0.1 Hz) breathing resonates with heart rate (coherent breathing), maximizing HF power — the optimal psychophysiological rhythm |
| `hrv.median_nn` | ±10% | Median NN interval (ms); normal 600–1000; < 600 = HR > 100 (tachycardia); 600–700 HR 86–100 (elevated); 700–900 HR 67–86 (normal); > 1000 HR < 60 (bradycardia/athlete); more robust than ibi_mean against occasional ectopic beats |
| `hrv.cvnn` | ±10% | NN coefficient of variation (SDNN/ibi_mean); normal 0.03–0.12; < 0.03 extremely rigid heart rate (may warrant attention); 0.03–0.06 low (stress/fatigue); 0.06–0.12 normal; > 0.15 high variability (high HRV / noise risk) |
| `hrv.cvsd` | ±10% | Coefficient of variation for successive differences (RMSSD/ibi_mean); normal 0.02–0.10; < 0.02 very low short-term variation (suppressed parasympathetic/overwork); 0.02–0.06 low; 0.06–0.10 normal; > 0.10 high (young/relaxed); > 0.15 high variability (noise risk) |
| `hrv.heart_rate` | ±2/±5/±8 bpm | HRV-derived heart rate (bpm); computed from NN intervals, may differ slightly from cardiac.hr |

<a id="bp"></a>

## `bp` Non-contact Blood Pressure

**Pack:** Core Physiology Pack

> Note: Estimated values with ~±8–12 mmHg error; not a substitute for medical devices; for trend reference only

| Field Path | Accuracy | Description |
| --- | --- | --- |
| `bp.sbp` | ±8–12 mmHg | Systolic BP (mmHg); < 90 triggers hypotension; 90–119 normal; 120–129 elevated/borderline (recheck recommended); 130–139 Stage 1 high; ≥ 140 Stage 2 high. Combined rule: if either SBP<90 or DBP<60, classify as low BP first; otherwise take the higher of the two grades ("higher-wins" principle) |
| `bp.dbp` | ±8–12 mmHg | Diastolic BP (mmHg); < 60 triggers hypotension; 60–79 ideal; 80–84 normal; 85–89 elevated/borderline; ≥ 90 Stage 1 high. Combined with sbp, the higher grade takes precedence |
| `bp.confidence` | — | BP estimation confidence 0–1; results are useful at ≥ 0.3; field may be absent when < 0.15 |
| `bp.map_est` | ±8–12 mmHg | Mean arterial pressure (mmHg); normal 70–100; < 60 is the shock threshold (requires urgent attention); > 110 suggests high perfusion pressure |
| `bp.pp_est` | ±8–12 mmHg | Pulse pressure (mmHg); normal 30–50; > 60 suggests arterial stiffness / aortic regurgitation; < 25 suggests hypovolemia / impaired cardiac output |
| `bp.features` | — | Pulse waveform features (ri, si, b_a, t_ratio) |

<a id="spo2"></a>

## `spo2` Non-contact SpO₂

**Pack:** Core Physiology Pack

> Note: Estimated values with ~±2–3% error; not a substitute for a finger pulse oximeter

| Field Path | Accuracy | Description |
| --- | --- | --- |
| `spo2.spo2` | ±2–3% | Blood oxygen saturation (%); 95–100% normal; 90–94% mild hypoxia (altitude/sleep apnea); < 90% hypoxemia |
| `spo2.confidence` | — | Estimation confidence 0–1 |
| `spo2.r_ratio` | — | Red/near-infrared ratio, SpO₂ intermediate value; normal population ~0.6–1.2 |

<a id="psych"></a>

## `psych` Psychological Composite Scores

**Pack:** Psychological Scores Pack · 30,000 Token/req

10 psychological metrics, all on a 0–100 scale. Score ranges: 85+ excellent; 75–85 good; 65–75 average; 60–65 slightly low; 50–60 notably low (only at high SQI). Four additional behavior-derived metrics are in the behavior module. Reference accuracy per dimension: ~±10%.

| Field Path | Accuracy | Description |
| --- | --- | --- |
| `psych.ms` | ±10% | Mental Stress; high HR + low SDNN → low score; 80+ low stress; < 60 high stress |
| `psych.mr` | ±10% | Mental Relaxation; Poincaré SD1/SD2 (short-term/long-term parasympathetic ratio); 80+ deep relaxation |
| `psych.mf` | ±10% | Mental Fatigue; HR/TP inversion, low TP + high HR → low score; 80+ energized |
| `psych.sq` | ±10% | Sleep Quality; combined frequency-domain and geometric features; decreases when LF or SD2 is low |
| `psych.mh` | ±10% | Mental Health; large Poincaré area = wide regulation range; reflects long-term psychological resilience |
| `psych.mb` | ±10% | Mental Balance; degree of sympatho-vagal imbalance from 1:1; 80+ highly balanced |
| `psych.con` | ±10% | Concentration; cognitive load causes HR↑, HRV↓, breathing rate changes; 80+ highly focused |
| `psych.mab` | ±10% | Memory / Autonomic Baseline; reflects neural "age effect" — younger, healthier baseline scores higher |
| `psych.adp` | ±10% | Adaptation; SDNN and RMSSD synergy; if either is low, recovery speed is limited |
| `psych.sra` | ±10% | Stress Resilience; maintains normal HR while preserving HRV reserve; 80+ can self-regulate under pressure |

<a id="emotion"></a>

## `emotion` Emotion Analysis

**Pack:** Surface Pack 5k · Deep Emotion Pack 20k Token

surface (facial expressions, can be faked) vs deep (face + rPPG fusion, spoof-resistant). Use deep for interviews/deception detection; use surface for UX testing. Recognition accuracy ~75% (same for surface and deep).

| Field Path | Accuracy | Description |
| --- | --- | --- |
| `emotion.surface.*` | 75% accuracy | 8-class surface emotion probability distribution summing to 1: Anger / Contempt / Disgust / Fear / Happiness / Neutral / Sadness / Surprise |
| `emotion.deep.*` | 75% accuracy | 8-class deep emotion (same field names); multimodal fusion, resistant to deliberate acting, more closely reflects true inner state |
| `emotion.dominant` | 75% accuracy | Dominant emotion name (highest probability among 8 classes) |
| `emotion.dominant_confidence` | — | Dominant emotion confidence; ≥ 0.5 reliable dominant; 0.25–0.5 tendency unclear; < 0.25 complex emotion or insufficient facial information |

<a id="face_au"></a>

## `face_au` Facial Action Units

**Pack:** Face AU Pack · 20,000 Token/req

20 fused AUs, range 0–1. General thresholds: 0–0.15 inactive; 0.15–0.40 mild; 0.40–0.65 moderate (usually visible); 0.65–1.0 strong. For gaze direction use behavior.gaze_yaw/gaze_pitch (multi-source fusion, higher accuracy). Recognition accuracy ~90%.

| Field Path | Accuracy | Description |
| --- | --- | --- |
| `brow_furrow` | 90% accuracy | Brow furrow; > 0.4 notable; anger (bilateral) or concentration (mild unilateral) |
| `brow_inner_raise` | 90% accuracy | Inner brow raise; > 0.3 sadness brow (linked with sadness_brow) |
| `brow_outer_raise` | 90% accuracy | Outer brow raise; > 0.3 surprise; < 0.1 with high brow_furrow = anger |
| `cheek_puff` | 90% accuracy | Cheek puff; > 0.4 holding breath or intentional puffing |
| `cheek_squint` | 90% accuracy | Cheek squint; > 0.3 Duchenne orbicularis contraction (genuine smile marker) |
| `eye_blink` | 90% accuracy | Eyelid closure during blink; > 0.35 blinking starts; > 0.8 fully closed (PERCLOS uses 0.8 threshold) |
| `eye_gaze_h` | 90% accuracy | Horizontal gaze (single source); positive = right; negative = left |
| `eye_gaze_v` | 90% accuracy | Vertical gaze (single source); positive = up; negative = down |
| `eye_squint` | 90% accuracy | Eye squint degree; concentration / genuine smile / bright light |
| `eye_wide` | 90% accuracy | Eye widening; > 0.4 fear or surprise |
| `jaw_open` | 90% accuracy | Jaw opening; > 0.3 speaking; > 0.6 wide open; combine with mouth_openness to detect speech |
| `jaw_lateral` | 90% accuracy | Jaw lateral shift; lateral chewing or speech deviation |
| `mouth_smile` | 90% accuracy | Corner lip raise; > 0.4 social smile; combine with cheek_squint to detect genuine smile |
| `mouth_smile_asym` | 90% accuracy | Smile asymmetry; > 0.15 contempt or unilateral facial nerve restriction |
| `mouth_frown` | 90% accuracy | Corner lip depression; sadness or disgust |
| `mouth_protrude` | 90% accuracy | Lip protrusion; pouting or kiss gesture |
| `mouth_stretch` | 90% accuracy | Lip stretch; wide smile or fear |
| `upper_lip_raise` | 90% accuracy | Upper lip raise; > 0.3 disgust |
| `nose_sneer` | 90% accuracy | Nose wrinkle; > 0.3 disgust or anger (combined with brow_furrow = aggression) |
| `tongue_out` | 90% accuracy | Tongue protrusion; > 0.5 clear tongue extension |

<a id="behavior"></a>

## `behavior` Behavioral Intent Indicators

**Pack:** Eye Intent Pack 20k · Psych Pack 30k · Health Pack 30k Token

17 eye-movement/intent metrics + 4 derived from psychological scores pack + 2 from facial health pack (bonus, no extra charge). Eye-movement intent metrics recognition accuracy ~80%.

| Field Path | Accuracy | Description |
| --- | --- | --- |
| `gaze_yaw` | 80% accuracy | Horizontal gaze offset (multi-source fusion); positive = right; negative = left; \|value\| > 1 marked deflection |
| `gaze_pitch` | 80% accuracy | Vertical gaze offset (multi-source fusion); positive = up; negative = down (e.g., looking down at phone) |
| `eye_aperture` | 80% accuracy | Eyelid openness 0–1; 0 = closed; 1 = normal open; > 1.0 wide-eyed (alarm/fear) |
| `perclos` | 80% accuracy | PERCLOS drowsiness index 0–1; > 0.15 driver monitoring fatigue threshold (SAE J2057); > 0.30 severe drowsiness |
| `blink_rate` | 80% accuracy | Blink frequency (blinks/min); normal 8–30; < 8 over-concentration or fatigue; > 30 nervous/irritated |
| `blink_close_ms` | 80% accuracy | Eyelid closing speed (ms); normal 70–100; > 150 heavy eyelids |
| `blink_dwell_ms` | 80% accuracy | Eye closed duration (ms); normal 50–150; > 200 fatigue/drowsiness indicator (Caffier 2003) |
| `blink_open_ms` | 80% accuracy | Eyelid opening speed (ms); normal 70–100; > 150 heavy eyelids |
| `gaze_stability` | 80% accuracy | Gaze stability 0–1; > 0.7 focused fixation; < 0.3 wandering gaze, distracted attention |
| `mouth_openness` | 80% accuracy | Mouth openness 0–1; > 0.15 speaking; > 0.5 wide open (surprise/yawn) |
| `talking` | 80% accuracy | Talking state (no audio needed); based on mouth motion variance |
| `clenching` | 80% accuracy | Jaw clenching tension 0–1; > 0.4 notable clenching (stress/pain) |
| `duchenne` | 80% accuracy | Genuine smile detection 0–1; > 0.3 authentic smile; < 0.1 with high smile = social fake smile |
| `sadness_brow` | 80% accuracy | Sadness brow; positive = inner brow raised + outer brow lowered (inverted-V); negative = furrowed |
| `aggression` | 80% accuracy | Aggression 0–1; > 0.5 brow furrow + nose sneer combined, strong aggression indicator |
| `confusion` | 80% accuracy | Confusion/thinking 0–1; > 0.2 asymmetric unilateral brow raise |
| `masking` | 80% accuracy | Emotional masking (binary); smile ≥ 0.6 and facial micro-motion ≤ 0.02 = 1 (forced smile) |
| `flow_state` | 80% accuracy | Flow state 0–1; 0.40×gaze_stability + 0.35×normal blink + 0.25×facial stillness; > 0.65 clear flow; < 0.30 difficulty entering flow |
| `emotional_masking` | 80% accuracy | Expression suppression 0–1; (smile−duchenne)×smile×micro-motion suppression; > 0.3 deliberate suppression detected; 0 no masking |
| `social_arousal` | 80% accuracy | Social arousal 0–1; talking(0.4) + smile(0.3) + facial motion(0.2) + gaze wander(0.1); > 0.6 highly engaged; < 0.2 passive withdrawal |
| `sleep_debt` | 80% accuracy | Sleep debt 0–1; 0.40×PERCLOS + 0.30×blink dwell + 0.15×slow blink + 0.15×low blink rate; > 0.5 severe; < 0.2 sufficient rest |
| `tremor_energy` | 80% accuracy | Micro-tremor energy 0–5; nasal tip displacement RMS at 3–7 Hz; normal < 0.1; > 0.5 notable; > 1.0 pathological (Parkinson's/severe anxiety) |
| `expression_distortion` | 80% accuracy | Expression distortion 0–1; 0.35×brow_furrow + 0.25×cheek + 0.20×nose_sneer + 0.20×eye_squint; > 0.4 notable distress/stress response |
| `smile_intensity` | 80% accuracy | Smile intensity 0–1; < 0.1 no smile; 0.1–0.3 slight smile; 0.3–0.6 clear smile; > 0.6 strong/broad smile; combine with duchenne to distinguish genuine vs social smile (bonus, no extra charge) |
| `smile_symmetry` | 80% accuracy | Smile asymmetry 0–1 (higher = more asymmetric); < 0.1 highly symmetric (natural, relaxed); 0.1–0.2 slight asymmetry (normal); > 0.3 notable asymmetry (contempt / unilateral nerve / deliberate expression) (bonus, no extra charge) |

<a id="appearance"></a>

## `appearance` Facial Attributes

**Pack:** Surface Pack · 5,000 Token/req

Age / Gender / Skin Tone (Fitzpatrick I–VI). Skin tone is mainly used to explain SQI differences: dark skin (V/VI) absorbs 520nm green light more strongly, resulting in SQI ~0.1–0.2 lower than I/II under the same conditions. Reference accuracy: age ±3.5 years; gender 96%; skin tone classification 80%.

| Field Path | Accuracy | Description |
| --- | --- | --- |
| `appearance.age.value` | ±3.5 yrs | Estimated age; reference accuracy ±3.5 years; can reach ±7 years with poor lighting or face angle |
| `appearance.age.range` | ±3.5 yrs | Age range string (e.g., "24-32"); narrower range = more typical facial features |
| `appearance.gender.label` | 96% accuracy | "Man" / "Woman" binary classification; no Unknown returned; closest category given even at low confidence; accuracy 96% |
| `appearance.gender.confidence` | — | Gender confidence; ≥ 0.85 reliable; 0.60–0.85 reference; < 0.60 suggest ignoring (heavy makeup/children/extreme angle) |
| `appearance.skin_tone.type` | 80% accuracy | Fitzpatrick type I–VI; I–III light skin highest SQI; IV typical East Asian (SQI ≥ 0.30 usable); V–VI dark skin SQI ~0.1–0.2 lower under same conditions; classification accuracy 80% |
| `appearance.skin_tone.ita` | — | ITA continuous chroma angle (CIE Lab); > 55→I; 41–55→II; 28–41→III; 10–28→IV; -30–10→V; < -30→VI |

<a id="liveness"></a>

## `liveness` Liveness Detection

**Pack:** Surface Pack

Determines real face vs photo/video/screen-replay attacks. rppg_sqi is the strongest anti-spoofing signal. Finance/identity verification: recommend liveness_score ≥ 0.8; general check-in: ≥ 0.5 sufficient. Low light, face < 20% of frame, masks/sunglasses all lower the score. Reference accuracy ~80% (standard capture conditions).

| Field Path | Accuracy | Description |
| --- | --- | --- |
| `liveness.is_live` | 80% accuracy | Final verdict (equivalent to liveness_score ≥ 0.5); business-facing LIVE/SPOOF label |
| `liveness.liveness_score` | 80% accuracy | Live-person confidence 0–1; ≥ 0.80 high-confidence live; 0.50–0.80 confident live; 0.45–0.50 borderline pass; 0.30–0.45 suspicious — re-record; < 0.30 high-probability attack (photo/screen replay) |
| `liveness.signals.blink_rate` | — | Blink rhythm score 0–1; photo = 0; real person 10–20 blinks/min with randomness; video replay scores lower |
| `liveness.signals.motion` | — | Head/face micro-motion 0–1; static photo ≈ 0; real person has breathing/balance/facial muscle micro-movement |
| `liveness.signals.rppg_sqi` | — | rPPG signal quality 0–1; real skin has blood-flow color micro-variation; screen/paper replay has none — strongest anti-spoofing signal |
| `liveness.signals.texture` | — | Texture authenticity 0–1; real skin has pores/fine lines; screen replay shows moiré patterns; printed photos have over-smooth texture |

<a id="billing"></a>

## `billing` Billing Fields

**Pack:** All endpoints

V3 uses tokens_deducted / remaining_tokens; V2/V1 legacy fields are points_deducted / remaining_points. Formula: Input Token = 30fps × duration(s) × 100; Output Token = sum of subscribed packs.

| Field Path | Accuracy | Description |
| --- | --- | --- |
| `data.tokens_deducted` | — | Tokens deducted for this request (V3; per-indicator granular billing, no rounding) |
| `data.remaining_tokens` | — | Remaining token balance after deduction (V3) |
| `data.points_deducted` | — | Tokens deducted for this request (V2/V1 legacy field name) |
| `data.remaining_points` | — | Remaining token balance (V2/V1 legacy field name) |

