// NOTE must be also eddited in specs.js
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
    },
    priority: ["jquery"]
});

require([
        "jquery",
        "react",
        "client/loginConn",
        "client/settings",
        "clientUI",
], function($, react, LoginConn, settings, clientUI) {
    var conn = new LoginConn(new WebSocket(settings.websocketURL));

    conn.on("connected", function() {
        console.log("connected to " + settings.websocketURL);
        $.getJSON("/actor/unique", function(actor) {
            conn.createActor("test"+actor.id, "test");
        });
    });

    conn.on("authFailed", function(name) {
        console.log("auth failed for", name);
    });

    var loginSuccess = function(actor, socket) {
        window.socket = socket;
        clientUI.takeoverDOM(socket, actor);
    };

    conn.on("loginSuccess", loginSuccess);
    conn.on("createSuccess", loginSuccess);
});
