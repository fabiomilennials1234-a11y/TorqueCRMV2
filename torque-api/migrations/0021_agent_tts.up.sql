-- =====================================================================
-- 0021_agent_tts.up.sql
-- S41 / Fase C.5 — F06 Copilot TTS (text-to-speech) config on agents.
--
-- Scope note: this migration adds only the agent-config surface
-- (tts_enabled + tts_voice_id). The outbound-message pipeline that
-- substitutes text for mp3 (worker on message.outbound) + storage
-- upload + provider quota decrement + org_quotas schema remain in a
-- follow-up sprint. Shipping the config first lets the Playground
-- preview the voice in real time without waiting for the full pipe.
--
-- Voice id is opaque provider string (ElevenLabs uses a ~20-char
-- alphanumeric like "21m00Tcm4TlvDq8ikWAM"). Short cap at 80 matches
-- our conservative text() policy on other provider id columns.
-- =====================================================================

ALTER TABLE agents
  ADD COLUMN tts_enabled  bool NOT NULL DEFAULT false,
  ADD COLUMN tts_voice_id text NULL CHECK (tts_voice_id IS NULL OR char_length(tts_voice_id) BETWEEN 2 AND 80);

COMMENT ON COLUMN agents.tts_enabled IS
  'S41 — when true + tts_voice_id non-null, outbound text messages are rendered as mp3 via the TTS provider. False keeps the agent text-only.';

COMMENT ON COLUMN agents.tts_voice_id IS
  'S41 — opaque provider voice identifier (e.g. ElevenLabs voice_id). NULL disables TTS regardless of tts_enabled.';
