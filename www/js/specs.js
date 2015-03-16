// NOTE must be also eddited in init.js
requirejs.config({
    baseUrl: "js",
    paths: {
        jquery:     "lib/jquery-1.11.2",
        underscore: "lib/underscore",
        react:      "lib/react",
        CAAT:       "lib/caat",
        app:        "app/app",
    },

    shim: {
        "underscore": {
            exports: function() {
                return this._.noConflict();
            },
        },
        
        "app": {
            deps:    ["client/settings"],
            exports: "gopherjsApplication",
            init:    function(settings) {
                return this.gopherjsApplication.initialize(settings);
            },
        },

        // Spec Framework
        "lib/jasmine.consolereporter": {
            deps: ["lib/jasmine"],
        },
        "lib/jasmine.htmlreporter": {
            deps: ["lib/jasmine"],
        },
    },

    priority: ["jquery"]
});

define("github.com/ghthor/engine/rpg2d/coord", ["app"], function(app) {
    return app.coord;
});

define("github.com/ghthor/aodd/game", ["app"], function(app) {
    return app.game;
});

define(["jquery",
       "underscore",

       // Specs
       "ui/canvas/terrain_map_spec",
       "ui/canvas/sprite/terrain_spec",

       // Spec Framework
       "lib/jasmine",
       "lib/jasmine.htmlreporter",
       "lib/jasmine.consolereporter",
], function($, _) {
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

            console.log("Running Specs...");
            jasmine.getEnv().execute();
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

            console.log("Running Specs...");
            jasmineEnv.execute();
        })
    };
});
