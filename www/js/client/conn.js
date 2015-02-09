define(["client/packet",
        "lib/minpubsub"
], function(Packet, pubSub) {
    // A connection is constructed with a websocket
    var Conn = function(socket) {
        var conn = this;
        conn.socket = socket;

        // websocket has connected
        socket.onopen = function() {
            conn.onmessage = handler.loginResp;
            conn.emit("connected");
        };

        // websocket has a message
        socket.onmessage = function(rawPacket) {
            conn.onmessage(Packet.Decode(rawPacket.data));
        };

        // websocket had an error
        socket.onerror = function() {
            console.log("error connecting to websocket");
            conn.emit("error");
        };

        var handler = {
            noop: function(packet) {
                console.log("noop packet processor", packet);
            },

            loginResp: function(packet) {
                var name;

                switch(packet.msg) {
                case "authFailed":
                    name = packet.payload;
                    conn.emit("authFailed", [name]);
                    break;

                case "actorDoesntExist":
                    // State change
                    //client.onmessage = handler.createActor;
                    conn.emit("actorDoesntExist", [JSON.parse(packet.payload)]);
                    break;

                case "loginSuccess":
                    // State change
                    //conn.onmessage = handler.bufferUpdates;
                    name = packet.payload;
                    conn.emit("loginSuccess", [name]);
                    break;

                default:
                    console.log("Unexpected packet during `login`", packet);
                }
            }
        };

        // Set the default packet handler
        conn.onmessage = handler.noop;

        return conn;
    };

    Conn.prototype = {
        attemptLogin: function(name, password) {
            var packet = Packet.JSON("login", {
                name: name,
                password: password
            });
            this.socket.send(Packet.Encode(packet));
        },
    };

    pubSub(Conn.prototype);

    return Conn;
});
