import Foundation

/// A thin Anthropic Messages API client over URLSession. CloudLens uses it to
/// turn the raw pulse findings into a short, human-readable summary for the
/// popover. There is no official Anthropic Swift SDK, so this calls the REST
/// endpoint directly. The API key is read from the Keychain, never stored in
/// the app bundle.
public struct AnthropicAgent: Sendable {
    public static let endpoint = URL(string: "https://api.anthropic.com/v1/messages")!
    public static let apiVersion = "2023-06-01"
    /// Default to the latest, most capable Claude model.
    public static let defaultModel = "claude-opus-4-8"

    let session: URLSession
    let model: String

    public init(session: URLSession = .shared, model: String = AnthropicAgent.defaultModel) {
        self.session = session
        self.model = model
    }

    public enum AgentError: Error, CustomStringConvertible {
        case noAPIKey
        case http(status: Int, body: String)
        case malformedResponse

        public var description: String {
            switch self {
            case .noAPIKey:
                return "Anthropic API key not set - add it in CloudLens settings."
            case .http(let status, let body):
                return "Anthropic API error \(status): \(body)"
            case .malformedResponse:
                return "Anthropic API returned an unexpected response shape."
            }
        }
    }

    /// Summarise the findings into one short paragraph for the popover header.
    /// `apiKey` is supplied by the caller (read from the Keychain) so this type
    /// stays free of storage concerns.
    public func summarise(_ findings: [Finding], apiKey: String) async throws -> String {
        guard !apiKey.isEmpty else { throw AgentError.noAPIKey }

        let prompt = Self.buildPrompt(findings)
        let body: [String: Any] = [
            "model": model,
            "max_tokens": 300,
            "messages": [
                ["role": "user", "content": prompt]
            ],
        ]

        var request = URLRequest(url: Self.endpoint)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.setValue(apiKey, forHTTPHeaderField: "x-api-key")
        request.setValue(Self.apiVersion, forHTTPHeaderField: "anthropic-version")
        request.httpBody = try JSONSerialization.data(withJSONObject: body)

        let (data, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse else {
            throw AgentError.malformedResponse
        }
        guard (200...299).contains(http.statusCode) else {
            throw AgentError.http(
                status: http.statusCode, body: String(decoding: data, as: UTF8.self))
        }
        return try Self.extractText(from: data)
    }

    /// Pull the first text block out of a Messages API response:
    /// `{ "content": [ { "type": "text", "text": "..." }, ... ] }`.
    static func extractText(from data: Data) throws -> String {
        guard
            let obj = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
            let content = obj["content"] as? [[String: Any]]
        else {
            throw AgentError.malformedResponse
        }
        let text = content
            .filter { ($0["type"] as? String) == "text" }
            .compactMap { $0["text"] as? String }
            .joined(separator: "\n")
        guard !text.isEmpty else { throw AgentError.malformedResponse }
        return text
    }

    static func buildPrompt(_ findings: [Finding]) -> String {
        if findings.isEmpty {
            return "Our cloud posture check found nothing notable. In one short, "
                + "matter-of-fact sentence, reassure an engineering lead that all is calm."
        }
        let lines = findings.map { f in
            "- [\(f.kind.rawValue)/\(f.severity == .attention ? "attention" : "info")] "
                + "\(f.title): \(f.detail)"
        }.joined(separator: "\n")
        return """
        You are a cloud sentinel for an engineering lead. Summarise these findings \
        in two or three short, matter-of-fact sentences. Lead with anything that \
        needs attention. No hyperbole.

        Findings:
        \(lines)
        """
    }
}
