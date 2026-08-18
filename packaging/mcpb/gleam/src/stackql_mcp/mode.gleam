//// Server mode. Default is ReadOnly; escalation is an explicit caller
//// opt-in, never a default (the embedding contract).

pub type Mode {
  ReadOnly
  Safe
  DeleteSafe
  FullAccess
}

/// The wire string stackql expects in `--mcp.config`.
pub fn to_string(mode: Mode) -> String {
  case mode {
    ReadOnly -> "read_only"
    Safe -> "safe"
    DeleteSafe -> "delete_safe"
    FullAccess -> "full_access"
  }
}
