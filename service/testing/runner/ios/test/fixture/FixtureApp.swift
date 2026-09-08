import SwiftUI

@main
struct EndlyMobileFixtureApp: App {
    @State private var count = 0

    var body: some Scene {
        WindowGroup {
            VStack(spacing: 20) {
                Text("Endly Mobile Runner")
                    .accessibilityIdentifier("status")
                Text("Count: \(count)")
                    .accessibilityIdentifier("count")
                Button("Increment") { count += 1 }
                    .accessibilityIdentifier("increment")
            }
            .padding()
        }
    }
}

