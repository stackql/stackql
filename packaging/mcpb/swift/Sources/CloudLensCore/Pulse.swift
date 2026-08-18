import Foundation
import StackQLMCP

/// A pulse runs one read_only check suite against the embedded StackQL server
/// and turns the rows into findings. Pulses are the unit the runner schedules.
public protocol Pulse: Sendable {
    var kind: PulseKind { get }
    /// Run the pulse against a connected server. Implementations should run
    /// only SELECT/metadata tools so they are safe under read_only.
    func run(_ server: StackQLServer) async -> PulseResult
}

/// Parse a StackQL tool text result into rows. StackQL returns query results
/// as a JSON array of objects; tolerate a leading/trailing log line by
/// extracting the outermost JSON array.
public enum RowParser {
    public static func rows(from text: String) -> [[String: Any]] {
        guard let json = outermostJSONArray(in: text),
              let data = json.data(using: .utf8),
              let arr = try? JSONSerialization.jsonObject(with: data) as? [[String: Any]]
        else {
            return []
        }
        return arr
    }

    /// Extract the substring from the first '[' to the last ']' inclusive, so
    /// surrounding diagnostics do not break parsing. Returns nil if absent.
    static func outermostJSONArray(in text: String) -> String? {
        guard let start = text.firstIndex(of: "["),
              let end = text.lastIndex(of: "]"),
              start < end
        else {
            return nil
        }
        return String(text[start...end])
    }
}

/// Helpers for reading typed values out of a parsed row.
extension Dictionary where Key == String, Value == Any {
    func string(_ key: String) -> String? {
        if let s = self[key] as? String { return s }
        if let n = self[key] as? NSNumber { return n.stringValue }
        return nil
    }

    func double(_ key: String) -> Double? {
        if let d = self[key] as? Double { return d }
        if let n = self[key] as? NSNumber { return n.doubleValue }
        if let s = self[key] as? String { return Double(s) }
        return nil
    }

    func int(_ key: String) -> Int? {
        if let i = self[key] as? Int { return i }
        if let n = self[key] as? NSNumber { return n.intValue }
        if let s = self[key] as? String { return Int(s) }
        return nil
    }
}
