"""Optional local STT. Requires faster-whisper; does not synthesize voices."""
import json
import sys

if len(sys.argv) != 2:
    raise SystemExit("Usage: transcribe_local.py audio_file")

try:
    from faster_whisper import WhisperModel
except ImportError:
    sys.stderr.write("faster-whisper not installed. Run 04_음성인식_설치.bat\n")
    raise SystemExit(2)

# Tiny model minimizes CPU memory; first run downloads public model weights.
model = WhisperModel("tiny", device="cpu", compute_type="int8")
segments, info = model.transcribe(sys.argv[1], language="ko", beam_size=3, vad_filter=True)
text = " ".join(s.text.strip() for s in segments).strip()
print(json.dumps({"text": text, "language": info.language,
                  "model": "faster-whisper tiny (local after download)",
                  "requires_review": True}, ensure_ascii=False))
