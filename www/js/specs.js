// NOTE must be also eddited in ui.js
requirejs.config({
    baseUrl: "js",
    paths: {
        jquery:     "lib/jquery-1.11.2",
        underscore: "lib/underscore",
        react:      "lib/react",
        CAAT:       "lib/caat"
    },
    shim: {
        "underscore": {
            exports: function() {
                return this._.noConflict();
            }
        },

        "lib/jasmine.consolereporter": {
            deps: ["lib/jasmine"]
        },
        "lib/jasmine.htmlreporter": {
            deps: ["lib/jasmine"]
        },
    },
    priority: ["jquery"]
});

define(["jquery",
       "underscore",

       // Specs
       "client/actors_spec",
       "client/terrainMap_spec",
       "client/inputState_spec",
       "client/updateBuffer_spec",
       "client/packet_test",
       "client/sprite/terrain_spec",

       // Spec Framework
       "lib/jasmine",
       "lib/jasmine.htmlreporter",
       "lib/jasmine.consolereporter"
], function($, _, describeActors) {
    var asyncInit = [describeActors];

    return {
        runConsoleReport: _.once(function() {
            var report = "";
            var consoleReporter = new jasmine.ConsoleReporter(function(str) {
                report += str;
            }, function() {
                console.log(report);
                console.log("Jasmine Testing Completed");
            });
            jasmine.getEnv().addReporter(consoleReporter);

            console.log("Initializing Specs...");

            var jasmineExecute = _.after(asyncInit.length, function() {
                console.log("Running Specs...");
                jasmine.getEnv().execute();
            });

            _.each(asyncInit, function(init) { init(jasmineExecute); });
        }),

        runHtmlReport: _.once(function() {
            $("head").prepend("<link rel='stylesheet' type='text/css' href='css/jasmine.css'>");
            var jasmineEnv = jasmine.getEnv();
            jasmineEnv.updateInterval = 1000;

            var report = "";
            var consoleReporter = new jasmine.ConsoleReporter(function(str) {
                report += str;
            }, function() {
                console.log(report);

                // Trigger the web server to shut down
                $.post("/specs/complete");
            });
            jasmineEnv.addReporter(consoleReporter);

            var htmlReporter = new jasmine.HtmlReporter();

            jasmineEnv.addReporter(htmlReporter);

            jasmineEnv.specFilter = function(spec) {
                return htmlReporter.specFilter(spec);
            };

            $(document).on("DOMSubtreeModified", function() {
                $(document).off("DOMSubtreeModified");
                $("#HTMLReporter").addClass("container");
            });

            console.log("Initializing Specs...");

            var jasmineExecute = _.after(asyncInit.length, function() {
                console.log("Running Specs...");
                jasmineEnv.execute();
            });

            _.each(asyncInit, function(init) { init(jasmineExecute); });
        })
    };
});
