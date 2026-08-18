import Foundation
import CloudLensCore
import UserNotifications
import os

/// Wraps UNUserNotificationCenter to post a native notification per new
/// finding. Per the design, the notification body includes the SQL behind the
/// finding so it is auditable from the notification itself.
@MainActor
final class Notifications {
    static let shared = Notifications()
    private let log = Logger(subsystem: "io.stackql.cloudlens", category: "notifications")

    /// Ask once for permission. Safe to call on every launch.
    func requestAuthorization() {
        UNUserNotificationCenter.current().requestAuthorization(
            options: [.alert, .sound]
        ) { [log] granted, error in
            if let error {
                log.error("notification auth error: \(error.localizedDescription)")
            } else {
                log.debug("notification auth granted=\(granted)")
            }
        }
    }

    /// Post one notification per new finding. Only attention-level findings
    /// notify; info-level changes update the popover silently.
    func post(_ findings: [Finding]) {
        let center = UNUserNotificationCenter.current()
        for finding in findings where finding.severity == .attention {
            let content = UNMutableNotificationContent()
            content.title = finding.title
            // Body carries the human detail plus the SQL behind the finding.
            content.body = "\(finding.detail)\n\nSQL:\n\(finding.sql)"
            content.sound = .default

            let request = UNNotificationRequest(
                identifier: finding.id,  // stable id de-dupes repeat notifications
                content: content,
                trigger: nil
            )
            center.add(request) { [log] error in
                if let error {
                    log.error("post notification failed: \(error.localizedDescription)")
                }
            }
        }
    }
}
