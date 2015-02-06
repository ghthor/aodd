var page = require("webpage").create();

var port = 45001;

// Provide a console.log for the test output
page.onConsoleMessage = function(msg) {
    console.log(msg);

    if (msg === "Jasmine Testing Completed") {
        phantom.exit();
    }
};

var runTest = function(failCount) {
    page.open("http://localhost:" + port, function(status) {
        if (status !== "success") {
            failCount++;
            if (failCount > 20) {
                console.log("Fatal: No webserver started@https://localhost:" + port, status);
                // page.open("https://localhost:"+port+"/testRunCompleted", function() {
                    phantom.exit();
                // });
            } else {
                console.log("Waiting for web server...");

                // Recurse till the webserver is alive
                runTest(failCount);
            }
        } else {

            // Call into the jasmine test code
            page.evaluate(function() {
                require(["main_test"], function(tester) {
                    tester.runConsoleReport();
                });
            });
        }
    });
};

runTest(0);
