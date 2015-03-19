// TODO This module is 100% unspecified
define(["lib/minpubsub",
       "CAAT",
], function(pubSub) {

    var ImageCache = function() {
        var imageCache = this;

        imageCache.load = function(images) {
            new CAAT.ImagePreloader().loadImages(images, function(loadCount, images) {
                if (loadCount === images.length) {
                    imageCache.images = images;
                    imageCache.emit("complete", [images]);
                }
            },
            function(imageError) {
                console.log("Error loading image: ", imageError);
            });
        };

        pubSub(imageCache);

        return this;
    };

    ImageCache.prototype = {
        loadDefault: function() {
            var spriteSheets = [
                {id: "female",        url: "asset/img/pokemon-female.png"},
                {id: "terrain_tiles", url: "asset/img/terrain-tiles-16x16x8.png"},
                {id: "rock",          url: "asset/rock.png"},
            ];
            this.load(spriteSheets);
        }
    };

    return ImageCache;
});
