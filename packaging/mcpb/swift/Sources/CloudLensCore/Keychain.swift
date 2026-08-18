import Foundation
import Security

/// A minimal Keychain-backed secret store for the Anthropic API key and
/// cloud provider credentials. Items are generic passwords scoped to the
/// app's service name, so they survive app updates and never touch disk in
/// plaintext.
public struct Keychain: Sendable {
    public let service: String

    public init(service: String = "io.stackql.cloudlens") {
        self.service = service
    }

    public enum KeychainError: Error, CustomStringConvertible {
        case unexpectedStatus(OSStatus)

        public var description: String {
            switch self {
            case .unexpectedStatus(let s):
                return "keychain error: \(SecCopyErrorMessageString(s, nil) as String? ?? "\(s)")"
            }
        }
    }

    /// Store or replace a secret for `account`. An empty value deletes it.
    public func set(_ value: String, for account: String) throws {
        if value.isEmpty {
            try remove(account)
            return
        }
        let data = Data(value.utf8)
        var query = baseQuery(account: account)
        let status = SecItemCopyMatching(query as CFDictionary, nil)
        if status == errSecSuccess {
            let attrs: [String: Any] = [kSecValueData as String: data]
            let upd = SecItemUpdate(query as CFDictionary, attrs as CFDictionary)
            guard upd == errSecSuccess else { throw KeychainError.unexpectedStatus(upd) }
        } else if status == errSecItemNotFound {
            query[kSecValueData as String] = data
            let add = SecItemAdd(query as CFDictionary, nil)
            guard add == errSecSuccess else { throw KeychainError.unexpectedStatus(add) }
        } else {
            throw KeychainError.unexpectedStatus(status)
        }
    }

    /// Fetch a secret, or nil if not set.
    public func get(_ account: String) throws -> String? {
        var query = baseQuery(account: account)
        query[kSecReturnData as String] = true
        query[kSecMatchLimit as String] = kSecMatchLimitOne
        var item: CFTypeRef?
        let status = SecItemCopyMatching(query as CFDictionary, &item)
        if status == errSecItemNotFound { return nil }
        guard status == errSecSuccess else { throw KeychainError.unexpectedStatus(status) }
        guard let data = item as? Data else { return nil }
        return String(decoding: data, as: UTF8.self)
    }

    /// Delete a secret. Missing item is not an error.
    public func remove(_ account: String) throws {
        let status = SecItemDelete(baseQuery(account: account) as CFDictionary)
        guard status == errSecSuccess || status == errSecItemNotFound else {
            throw KeychainError.unexpectedStatus(status)
        }
    }

    private func baseQuery(account: String) -> [String: Any] {
        [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
        ]
    }
}

/// Well-known Keychain account names CloudLens uses.
public enum SecretKey {
    public static let anthropicAPIKey = "anthropic-api-key"
}
