// TODO This module is 100% unspecified
define(["client/packet",
       "client/imageCache",
       "client/world",
       "client/inputState",
       "client/chat",
       "client/updateBuffer",
       "lib/minpubsub",
       "CAAT"
], function(Packet, ImageCache, World, InputState, Chat, UpdateBuffer, pubSub) {
    var Client = function(socket, actorEntity) {
        var client = this;

        // Initially we're going to buffer all the updates
        var updateBuffer = new UpdateBuffer();
        var onmessage = function(packet) {
            if (packet.msg === "update") {
                updateBuffer.merge(JSON.parse(packet.payload));
            } else {
                console.log(packet);
            }
        };

        // Set all the callbacks on the socket object
        socket.onconnect = function() {
            throw "unexpected `onconnect` call";
        };

        socket.onmessage = function(rawPacket) {
            onmessage(Packet.Decode(rawPacket.data));
        };

        socket.onerror = function() {
            console.log("error connecting to websocket");
        };

        var startRendering = function(imageCache) {
            CAAT.DEBUG = 1;

            // Create the Director and Scene
            var director = new CAAT.Director().
                initialize(800, 600).
                setImagesCache(imageCache);
            var scene = director.createScene().setFillStyle("#c0c0c0");

            // Create a new World that we will display
            var world = new World(director, scene, actorEntity);
            world.update(updateBuffer.merged());

            // Create a new input state manager
            var inputState = new InputState(socket);

            var chat = (function() {
                var eventPublisher = client;
                return new Chat(socket, eventPublisher, function(entityId) {
                    return world.entityForId(entityId);
                });
            }());

            // Set the new packet handler
            onmessage = function(packet) {
                if (packet.msg === "update") {
                    var worldState = JSON.parse(packet.payload);
                    world.update(worldState);
                    inputState.update(worldState.time);
                    chat.update(worldState);
                } else {
                    console.log(packet);
                }
            };

            client.emit("ready", [director.canvas, inputState, chat]);

            CAAT.loop();
        };

        // Create an image cache of all the game assests
        var imageCache = new ImageCache();

        // Set a listener for the assets being loaded
        imageCache.on("complete", startRendering);

        // TODO Experiment loading images at different points
        // * Before User Logged in - Fastest, but uses more bandwidth if the user leaves before playing
        // * Right here - Slowest load since we'll have many WorldStates before we're ready to render
        imageCache.loadDefault();

        return this;
    };

    // Extend Client to be a message publisher generator
    pubSub(Client.prototype);

    return Client;
});
