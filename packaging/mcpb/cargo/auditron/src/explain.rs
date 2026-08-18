//! Finding explanation via an agent. Claude (Messages API over plain HTTP)
//! is the default implementation; the trait keeps it pluggable.

use anyhow::{anyhow, Context, Result};

use crate::engine::ControlResult;
use crate::pack::Control;

/// Rows included in the prompt at most; evidence packs carry the full set.
const PROMPT_ROW_SAMPLE: usize = 20;

pub trait Explainer: Send + Sync {
    /// Explain a finding and draft remediation steps. Blocking.
    fn explain(&self, control: &Control, result: &ControlResult) -> Result<String>;
}

/// Claude via the Messages API (no official Rust SDK; raw HTTP via ureq).
pub struct ClaudeExplainer {
    api_key: String,
    model: String,
}

impl ClaudeExplainer {
    /// Returns None when ANTHROPIC_API_KEY is unset; the TUI then shows the
    /// SQL and a hint instead of an explanation.
    pub fn from_env() -> Option<Self> {
        let api_key = std::env::var("ANTHROPIC_API_KEY")
            .ok()
            .filter(|k| !k.is_empty())?;
        let model =
            std::env::var("ANTHROPIC_MODEL").unwrap_or_else(|_| "claude-opus-4-8".to_string());
        Some(ClaudeExplainer { api_key, model })
    }
}

impl Explainer for ClaudeExplainer {
    fn explain(&self, control: &Control, result: &ControlResult) -> Result<String> {
        let sample: Vec<_> = result.rows.iter().take(PROMPT_ROW_SAMPLE).collect();
        let user_message = format!(
            "Control: {} - {}\n\nDescription: {}\n\nSQL that produced the finding:\n{}\n\n\
             Status: {:?} ({} rows{}).\n\nFinding rows (sample of {}):\n{}\n{}",
            control.id,
            control.title,
            control.description,
            result.sql,
            result.status,
            result.rows.len(),
            result
                .error
                .as_deref()
                .map(|e| format!(", error: {e}"))
                .unwrap_or_default(),
            sample.len(),
            serde_json::to_string_pretty(&sample)?,
            control
                .remediation
                .as_deref()
                .map(|r| format!("\nPack-suggested remediation: {r}"))
                .unwrap_or_default(),
        );

        let body = serde_json::json!({
            "model": self.model,
            "max_tokens": 16000,
            "thinking": {"type": "adaptive"},
            "system": "You are a compliance copilot embedded in auditron, a terminal \
                audit tool. The user is a compliance engineer reviewing a failed or \
                errored control from a point-in-time scan run over StackQL (cloud APIs \
                exposed as SQL). Explain what the finding means, why it matters, and \
                give concrete, numbered remediation steps. Be matter-of-fact and \
                concise; plain text only, no markdown headings.",
            "messages": [{"role": "user", "content": user_message}],
        });

        let mut response = ureq::post("https://api.anthropic.com/v1/messages")
            .header("x-api-key", &self.api_key)
            .header("anthropic-version", "2023-06-01")
            .send_json(&body)
            .context("calling the Claude Messages API")?;
        let parsed: serde_json::Value = response
            .body_mut()
            .read_json()
            .context("parsing Messages API response")?;

        if parsed["stop_reason"] == "refusal" {
            return Err(anyhow!("the model declined to answer this request"));
        }
        // Content is a block array; thinking blocks precede text blocks.
        let text: String = parsed["content"]
            .as_array()
            .ok_or_else(|| anyhow!("unexpected response shape: {parsed}"))?
            .iter()
            .filter(|block| block["type"] == "text")
            .filter_map(|block| block["text"].as_str())
            .collect::<Vec<_>>()
            .join("\n");
        if text.is_empty() {
            return Err(anyhow!("model returned no text content: {parsed}"));
        }
        Ok(text)
    }
}
