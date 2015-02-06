define(["lib/minpubsub",
       "CAAT"
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
                {id: "female",        url: "img/pokemon-female.png"},
                {id: "terrain_tiles", url: "img/terrain-tiles-16x16x8.png"}
            ];
            this.load(spriteSheets);
        }
    };

    return ImageCache;
});
