var page = require("webpage").create();

var port = 45001;

// Provide a console.log for the test output
page.onConsoleMessage = function(msg) {
    console.log(msg);

    if (msg === "Jasmine Testing Completed") {
        phantom.exit();
    }
};

(function() {
    page.open("http://localhost:" + port, function() {});
}());
