// TODO This module is 100% unspecified
// TODO Fix these import paths
define(["client/imageCache",
       "client/world",
       "client/inputState",
       "client/chat",
       "lib/minpubsub",
       "CAAT",
], function(ImageCache, World, InputState, Chat, pubsub) {
    var Canvas = function() {
        var canvas = this;

        // TODO fix chat's parameter and remove this mock
        var socket = {
            "send": function() {},
        };

        var startRendering = function(imageCache) {
            CAAT.DEBUG = 1;

            // Create the Director and Scene
            var director = new CAAT.Director().
                initialize(800, 600).
                setImagesCache(imageCache);
            var scene = director.createScene().setFillStyle("#c0c0c0");

            var world = new World(director, scene);

            // Create a new input state manager
            var inputState = new InputState(socket);

            var chat = (function() {
                var eventPublisher = canvas;
                return new Chat(socket, eventPublisher, function(entityId) {
                    return world.entityForId(entityId);
                });
            }());

            canvas.on("update", function(worldStateDiff) {
                    world.update(worldStateDiff);
                    inputState.update(worldStateDiff.time);
                    chat.update(worldStateDiff);
            });

            canvas.emit("ready", [director.canvas, inputState, chat]);

            CAAT.loop();
        };

        // Create an image cache of all the game assests
        var imageCache = new ImageCache();
        imageCache.on("complete", startRendering);
        imageCache.loadDefault();

        return this;
    };

    // Extend Canvas to be a message publisher generator
    pubsub(Canvas.prototype);

    return Canvas;
});
