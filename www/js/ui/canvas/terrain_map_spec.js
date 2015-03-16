define(["ui/canvas/terrain_map",
       "underscore",
       "lib/jasmine"
], function(TerrainMap, _) {

    var MockSprite = function() {};
    MockSprite.prototype = {
        makeGrassTile: function(cell) {
            return {
                paint: jasmine.createSpy("paint"),
                type: "G",
                cell: cell,
            };
        },

        makeDirtTile: function(cell) {
            return {
                paint: jasmine.createSpy("paint"),
                type: "D",
                cell: cell,
            };
        },

        makeRockTile: function(cell) {
            return {
                paint: jasmine.createSpy("paint"),
                type: "R",
                cell: cell,
            };
        },
    };

    var newTerrainMap = function(def) {
        var canvas = {
            getContext: function() {
                return {
                    drawImage: jasmine.createSpy("drawImage"),
                };
            },
        };

        var terrainMap = new TerrainMap(def, canvas, 10);
        return terrainMap;
    };

    describe("a terrain map", function() {
        var oldSprite;
        beforeEach(function() {
            oldSprite = TerrainMap.sprite;
            TerrainMap.sprite = new MockSprite();

            this.addMatchers({
                toHaveTerrain: function(terrain) {
                    var row = this.actual;

                    row = _.pluck(row, "type").join("");
                    this.message = function() {
                        return "Expected `" + row + "` to equal `" + terrain + "`";
                    };
                    return row === terrain;
                }
            });
        });

        afterEach(function() {
            TerrainMap.sprite = oldSprite;
        });

        describe("knows when it is", function() {
            it("a full map", function() {
                var map = newTerrainMap({
                    Bounds: {
                        TopL: {X:-1, Y: 1},
                        BotR: {X: 1, Y:-1},
                    },
                    Terrain: "\nDGG\nGDG\nGGD\n",
                });
                expect(map.isSlice()).toBe(false);
            });

            it("a vertical slice", function() {
                var map = newTerrainMap({
                    Bounds: {
                        TopL: {X:-1, Y:1},
                        BotR: {X:-1, Y:-1},
                    },
                    Terrain: "\nD\nG\nG\n",
                });
                expect(map.isSlice()).toBe(true);
            });

            it("a horizontal slice", function() {
                var map = newTerrainMap({
                    Bounds: {
                        TopL: {X:-1, Y:1},
                        BotR: {X:1, Y:1}
                    },
                    Terrain: "\nDGG\n",
                });
                expect(map.isSlice()).toBe(true);
            });
        });

        it("parses terrain defination string into an array", function() {
            var bounds = {
                TopL: {X:-1, Y: 1},
                BotR: {X: 1, Y:-1},
            };

            var map = newTerrainMap({
                Bounds: bounds,
                Terrain: "\nDGG\nRDG\nRRD\n",
            });
            // height == 3
            expect(map.sprites.length).toBe(3);

            // Check sprite types
            expect(map.sprites[0]).toHaveTerrain("DGG");
            expect(map.sprites[1]).toHaveTerrain("RDG");
            expect(map.sprites[2]).toHaveTerrain("RRD");
        });

        xit("adds all the actors to the screen", function() {
            var bounds = {
                TopL: {X:-1, Y: 1},
                BotR: {X: 1, Y:-1},
            };

            var map = newTerrainMap({
                Bounds: bounds,
                Terrain: "\nDGG\nRDG\nRRD\n",
            });

            expect(map.addTileToScreen.calls.length).toBe(9);
        });

        it("can find it's center", function() {
            var bounds = {
                TopL: {X:-1, Y: 1},
                BotR: {X: 1, Y:-1},
            };

            var map = newTerrainMap({
                Bounds: bounds,
                Terrain: "\nDGG\nRDG\nRRD\n",
            });

            expect(map.center()).toEqual({X: 0, Y: 0});
        });

        describe("can be merged with", function() {
            var map;
            beforeEach(function() {
                map = newTerrainMap({
                    Bounds: {
                        TopL: {X:-1, Y: 1},
                        BotR: {X: 1, Y:-1},
                    },
                    Terrain: "\nDGG\nGDG\nGGD\n",
                });

                this.addMatchers({
                    toHaveBounds: function(expected) {
                        var bounds = this.actual.Bounds;

                        this.message = function() {
                            return "Expected " + bounds + "to be " + expected;
                        };

                        return bounds.TopL.X === expected.TopL.X &&
                            bounds.TopL.Y === expected.TopL.Y &&
                            bounds.BotR.X === expected.BotR.X &&
                            bounds.BotR.Y === expected.BotR.Y;
                    }
                });
                expect(map).toHaveBounds(map.Bounds);
            });

            it("a horizontal map slice to the north", function() {
                var slice = newTerrainMap({
                    Bounds: {
                        TopL: {X:-1, Y: 2},
                        BotR: {X: 1, Y: 2},
                    },
                    Terrain: "\nRRR\n",
                });

                expect(function() { map = map.merge(slice); }).not.toThrow();

                var bounds = {
                    TopL: {X: -1, Y: 2},
                    BotR: {X:  1, Y: 0},
                };

                expect(map).toHaveBounds(bounds);

                // height == 3
                expect(map.sprites.length).toBe(3);

                // Check sprite types
                expect(map.sprites[0]).toHaveTerrain("RRR");
                expect(map.sprites[1]).toHaveTerrain("DGG");
                expect(map.sprites[2]).toHaveTerrain("GDG");
            });

            it("a horizontal map slice to the south", function() {
                var slice = newTerrainMap({
                    Bounds: {
                        TopL: {X:-1, Y:-2},
                        BotR: {X: 1, Y:-2},
                    },
                    Terrain: "\nRRR\n",
                });

                expect(function() { map = map.merge(slice); }).not.toThrow();

                var bounds = {
                    TopL: {X: -1, Y: 0},
                    BotR: {X:  1, Y: -2},
                };

                expect(map).toHaveBounds(bounds);

                // height == 3
                expect(map.sprites.length).toBe(3);

                // Check sprite types
                expect(map.sprites[0]).toHaveTerrain("GDG");
                expect(map.sprites[1]).toHaveTerrain("GGD");
                expect(map.sprites[2]).toHaveTerrain("RRR");
            });

            it("a vertical map slice to the west", function() {
                var slice = newTerrainMap({
                    Bounds: {
                        TopL: {X:-2, Y: 1},
                        BotR: {X:-2, Y:-1},
                    },
                    Terrain: "\nR\nR\nR\n",
                });

                expect(function() { map = map.merge(slice); }).not.toThrow();

                // The bounds must reflect the merge
                var bounds = {
                    TopL: {X: -2, Y: 1},
                    BotR: {X:  0, Y: -1},
                };

                expect(map).toHaveBounds(bounds);

                // height == 3
                expect(map.sprites.length).toBe(3);

                // Check sprite types
                expect(map.sprites[0]).toHaveTerrain("RDG");
                expect(map.sprites[1]).toHaveTerrain("RGD");
                expect(map.sprites[2]).toHaveTerrain("RGG");
            });

            it("a vertical map slice to the east", function() {
                var slice = newTerrainMap({
                    Bounds: {
                        TopL: {X:2, Y: 1},
                        BotR: {X:2, Y:-1},
                    },
                    Terrain: "\nR\nR\nR\n",
                });

                expect(function() { map = map.merge(slice); }).not.toThrow();

                var bounds = {
                    TopL: {X: 0, Y: 1},
                    BotR: {X: 2, Y: -1},
                };

                expect(map).toHaveBounds(bounds);

                // height == 3
                expect(map.sprites.length).toBe(3);

                // Check sprite types
                expect(map.sprites[0]).toHaveTerrain("GGR");
                expect(map.sprites[1]).toHaveTerrain("DGR");
                expect(map.sprites[2]).toHaveTerrain("GDR");
            });
        });
    });
});
