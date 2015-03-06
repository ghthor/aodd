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
   "lib/minpubsub",
], function(app, pubsub) {

    // An object that will respond to events published
    // to it. This enables it to handle everything that
    // likes to live a JS only world, like react and the
    // rendering library.
    var UI = function() {
        var ui = this;

        // Respond to an event
        ui.on(app.EV_TICK, function(e) {
            console.log(e.time);
        });
    };

    pubsub(UI.prototype);

    app.setTickerUI(new UI());
});
