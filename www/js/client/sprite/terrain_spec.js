define(["client/sprite/terrain",
       "underscore",
       "lib/jasmine"
], function(Tile) {
    describe("when creating a sprite for", function() {
        describe("a grass tile", function() {
            it("the random sprite index is a valid sprite index", function() {
                var tile = new Tile();
                spyOn(tile, "makeTile");

                for (var i = 0; i < 100; i++) {
                    tile.makeGrassTile();
                    var spriteIndex = tile.makeTile.calls[i].args[0];

                    expect(Tile.indexs.grass).toContain(spriteIndex);
                }
            });
        });
    });
});
