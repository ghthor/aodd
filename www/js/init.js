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
            }
        },
        
        "app": {
            deps: ["client/settings"],
            exports: "gopherjsApplication",
            init: function(settings) {
                return this.gopherjsApplication.initialize(settings);
            },
        },
    },
    priority: ["jquery"],
});

define("github.com/ghthor/aodd/game", ["app"], function(app) {
    return app.game;
});

require([
   "app",
   "ui/login",
], function(app, LoginUI) {
    var container = document.getElementById("client");

    app.dial(new LoginUI(container));
});
