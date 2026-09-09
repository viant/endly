import XCTest

final class FixtureAppFailureTests: XCTestCase {
    func testPassingControl() {
        XCTAssertEqual(2 + 2, 4)
    }

    func testIntentionalFailure() {
        XCTAssertEqual("actual", "expected", "intentional Endly XCTest failure")
    }
}
