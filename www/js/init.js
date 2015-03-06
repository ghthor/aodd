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
            exports: "gopherjsApplication",
        },
    },
    priority: ["jquery"],
});

require([
   "app",
   "ui/login",
], function(app, LoginUI) {

    var loginConn = {
        attemptLogin: app.attemptLogin,
        createActor: app.createActor,
    };

    app.setTickerUI(new LoginUI(loginConn));
});
