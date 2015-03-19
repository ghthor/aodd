// TODO This module is 100% unspecified
// TODO Fix these import paths
define(["jquery",
       "app",
       "ui/canvas/image_cache",
       "ui/canvas/world",
       "ui/canvas/terrain_map",
       "lib/minpubsub",
       "CAAT",
], function($, app, ImageCache, World, TerrainMap, pubsub) {
    var Canvas = function(client) {
        var canvas = this;

        var startRendering = function(imageCache) {
            CAAT.DEBUG = 1;

            // Create the Director and Scene
            var director = new CAAT.Director().
                initialize(800, 600).
                setImagesCache(imageCache);
            var scene = director.createScene().setFillStyle("#c0c0c0");

            TerrainMap.initialize(director);

            var world = new World(director, scene);

            client.on(app.EV_ERROR, function(error) {
                console.log(error);
            });

            client.on(app.EV_RECV_INITIAL_STATE, function(entity, worldState, terrainMap) {
                var canvas = document.createElement("canvas");
                $(canvas).css({
                    display:  "none",
                    position: "absolute",
                    top:      -9999,
                    left:     -9999,
                });
                document.body.appendChild(canvas);

                // TODO Factor out tileSz of 16 to settings
                var map = new TerrainMap(terrainMap, canvas, 16, client);

                world.initialize(entity, worldState, {
                    map:    map,
                    canvas: canvas,
                });
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
