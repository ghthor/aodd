// TODO This module is 100% unspecified
define(["client/packet",
        "lib/minpubsub"
], function(Packet, pubSub) {
    // A connection is constructed with a websocket
    var Conn = function(socket) {
        var conn = this;

        var onmessage;

        // websocket has connected
        socket.onopen = function() {
            conn.emit("connected");
        };

        // websocket has a message
        socket.onmessage = function(rawPacket) {
            onmessage(Packet.Decode(rawPacket.data));
        };

        // websocket had an error
        socket.onerror = function() {
            console.log("error connecting to websocket");
            conn.emit("error");
        };

        var handlers = {
            noop: function(packet) {
                console.log("noop packet processor", packet);
            },

            login: function(packet) {
                var name;

                switch(packet.msg) {
                case "authFailed":
                    name = packet.payload;
                    conn.emit("authFailed", [name]);
                    break;

                case "actorDoesntExist":
                    conn.emit("actorDoesntExist", [JSON.parse(packet.payload)]);
                    break;

                case "loginSuccess":
                    name = packet.payload;

                    // Unset the handler
                    onmessage = handlers.noop;

                    // Pass off the socket to the outside world
                    conn.emit("loginSuccess", [name, socket]);
                    break;

                default:
                    console.log("Unexpected packet in response to `login` request", packet);
                }
            },

            create: function(packet) {
                var name;

                switch (packet.msg) {
                case "actorAlreadyExists":
                    name = packet.payload;
                    conn.emit("actorAlreadyExists", [name]);
                    break;

                case "createSuccess":
                    name = packet.payload;

                    // Unset the handler
                    onmessage = handlers.noop;

                    // Pass off the socket to the outside world
                    conn.emit("createSuccess", [name, socket]);
                    break;

                default:
                    console.log("Unexpected packet in response to `create` request", packet);
                }
            }
        };

        // Set the default packet handler
        onmessage = handlers.noop;

        // Create methods to send requests
        conn.attemptLogin = function(name, password) {
            // Set the response handler
            onmessage = handlers.login;

            var packet = Packet.JSON("login", {
                name: name,
                password: password
            });

            socket.send(Packet.Encode(packet));
        };

        conn.createActor = function(name, password) {
            // Set response handler
            onmessage = handlers.create;

            var packet = Packet.JSON("create", {
                name: name,
                password: password
            });

            socket.send(Packet.Encode(packet));
        };

        return conn;
    };

    pubSub(Conn.prototype);

    return Conn;
});
