var page = require("webpage").create();

var port = 45001;

// Provide a console.log for the test output
page.onConsoleMessage = function(msg) {
    console.log(msg);

    if (msg === "Jasmine Testing Completed") {
        phantom.exit();
    }
};

var runTest = function() {
    page.open("http://localhost:" + port, function() {
        // Call into the jasmine test code
        page.evaluate(function() {
            require(["main_test"], function(tester) {
                tester.runConsoleReport();
            });
        });
    });
};

runTest();
