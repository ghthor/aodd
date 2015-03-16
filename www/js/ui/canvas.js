// TODO This module is 100% unspecified
// TODO Fix these import paths
define(["app",
       "ui/canvas/image_cache",
       "ui/canvas/world",
       "lib/minpubsub",
       "CAAT",
], function(app, ImageCache, World, pubsub) {
    var Canvas = function(client) {
        var canvas = this;

        // TODO fix chat's parameter and remove this mock
        var startRendering = function(imageCache) {
            CAAT.DEBUG = 1;

            // Create the Director and Scene
            var director = new CAAT.Director().
                initialize(800, 600).
                setImagesCache(imageCache);
            var scene = director.createScene().setFillStyle("#c0c0c0");

            var world = new World(director, scene);

            client.on(app.EV_ERROR, function(error) {
                console.log(error);
            });

            client.on(app.EV_RECV_INITIAL_STATE, function(entity, worldState) {
                world.initialize(entity, worldState);
            });

            client.on(app.EV_RECV_UPDATE, function(worldStateDiff) {
                world.update(worldStateDiff);
            });

            canvas.emit("ready", [director.canvas]);

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
