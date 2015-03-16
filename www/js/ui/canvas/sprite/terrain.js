define(["CAAT"], function() {
    var makeImage = function(director) {
        return new CAAT.SpriteImage().initialize(director.getImage("terrain_tiles"), 2, 4);
    };

    var Tile = function(image) {
        this.makeTile = function(index, cell) {
            var img = image.getRef();

            img.cell = cell;
            img.paint = function(ctx, x, y) {
                img.paintTile(ctx, index, x, y);
            };

            return img;
        };
    };

    var indexs = {
        grass: [0,1,2,3],
        dirt: 4,
        rock: 5,
    };

    var getRandomInt = function(min, max) {
        return Math.floor(Math.random() * (max - min)) + min;
    };

    var makeGrassTile = function(cell) {
        var i = getRandomInt(0, indexs.grass.length);
        return this.makeTile(indexs.grass[i], cell);
    };

    var makeDirtTile = function(cell) {
        return this.makeTile(indexs.dirt, cell);
    };

    var makeRockTile = function(cell) {
        return this.makeTile(indexs.rock, cell);
    };

    Tile.makeImage = makeImage;
    Tile.indexs    = indexs;

    // interface
    Tile.prototype = {
        makeGrassTile: makeGrassTile,
        makeDirtTile:  makeDirtTile,
        makeRockTile:  makeRockTile,
    };

    return Tile;
});
