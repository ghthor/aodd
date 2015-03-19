define(["jquery",
        "app",
        "ui/client",
        "lib/minpubsub",
], function($, app, Client, pubsub) {
    var Login = function(container) {
        var login = this;

        login.on(app.EV_CONNECTED, function(conn) {
            login.on(app.EV_ACTOR_EXISTS, function() {
                console.log("actor", name, "exists");
            });

            login.on(app.EV_CREATE_SUCCESS, function(actor, loggedInConn) {
                var ui = new Client(container, loggedInConn);
                ui.render();
            });

            $.getJSON("/actor/unique", function(actor) {
                conn.createActor("test"+actor.id, "test");
            });
        });

        return this;
    };

    pubsub(Login.prototype);

    return Login;
});
