define(["client/terrainMap",
       "underscore",
       "lib/jasmine"
], function(TerrainMap, _) {

    var MockSprite = function() {};
    MockSprite.prototype = {
        makeGrassTile: function(cell) {
            return {
                paint: jasmine.createSpy("createSpy"),
                type: "G",
                cell: cell
            };
        },
        makeDirtTile:  function(cell) {
            return {
                paint: jasmine.createSpy("createSpy"),
                type: "D",
                cell: cell
            };
        },
        makeRockTile:  function(cell) {
            return {
                paint: jasmine.createSpy("createSpy"),
                type: "R",
                cell: cell
            };
        }
    };
    var newTerrainMap = function(def) {
        var canvas = {
            getContext: function() {
                return {
                    drawImage: jasmine.createSpy("drawImage")
                };
            }
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
                    bounds: {
                        tl: {x:-1, y: 1},
                        br: {x: 1, y:-1}
                    },
                    terrain: "\nDGG\nGDG\nGGD\n"
                });
                expect(map.isSlice()).toBe(false);
            });

            it("a vertical slice", function() {
                var map = newTerrainMap({
                    bounds: {
                        tl: {x:-1, y:1},
                        br: {x:-1, y:-1}
                    },
                    terrain: "\nD\nG\nG\n"
                });
                expect(map.isSlice()).toBe(true);
            });

            it("a horizontal slice", function() {
                var map = newTerrainMap({
                    bounds: {
                        tl: {x:-1, y:1},
                        br: {x:1, y:1}
                    },
                    terrain: "\nDGG\n"
                });
                expect(map.isSlice()).toBe(true);
            });
        });

        it("parses terrain defination string into an array", function() {
            var bounds = {
                tl: {x:-1, y: 1},
                br: {x: 1, y:-1}
            };

            var map = newTerrainMap({
                bounds: bounds,
                terrain: "\nDGG\nRDG\nRRD\n"
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
                tl: {x:-1, y: 1},
                br: {x: 1, y:-1}
            };

            var map = newTerrainMap({
                bounds: bounds,
                terrain: "\nDGG\nRDG\nRRD\n"
            });

            expect(map.addTileToScreen.calls.length).toBe(9);
        });

        it("can find it's center", function() {
            var bounds = {
                tl: {x:-1, y: 1},
                br: {x: 1, y:-1}
            };

            var map = newTerrainMap({
                bounds: bounds,
                terrain: "\nDGG\nRDG\nRRD\n"
            });

            expect(map.center()).toEqual({x: 0, y: 0});
        });

        describe("can be merged with", function() {
            var map;
            beforeEach(function() {
                map = newTerrainMap({
                    bounds: {
                        tl: {x:-1, y: 1},
                        br: {x: 1, y:-1}
                    },
                    terrain: "\nDGG\nGDG\nGGD\n"
                });

                this.addMatchers({
                    toHaveBounds: function(expected) {
                        var bounds = this.actual.bounds;

                        this.message = function() {
                            return "Expected " + bounds + "to be " + expected;
                        };

                        return bounds.tl.x === expected.tl.x &&
                            bounds.tl.y === expected.tl.y &&
                            bounds.br.x === expected.br.x &&
                            bounds.br.y === expected.br.y;
                    }
                });
                expect(map).toHaveBounds(map.bounds);
            });

            it("a horizontal map slice to the north", function() {
                var slice = newTerrainMap({
                    bounds: {
                        tl: {x:-1, y: 2},
                        br: {x: 1, y: 2}
                    },
                    terrain: "\nRRR\n"
                });

                expect(function() { map = map.merge(slice); }).not.toThrow();

                var bounds = {
                    tl: {x: -1, y: 2},
                    br: {x:  1, y: 0}
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
                    bounds: {
                        tl: {x:-1, y:-2},
                        br: {x: 1, y:-2}
                    },
                    terrain: "\nRRR\n"
                });

                expect(function() { map = map.merge(slice); }).not.toThrow();

                var bounds = {
                    tl: {x: -1, y: 0},
                    br: {x:  1, y: -2}
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
                    bounds: {
                        tl: {x:-2, y: 1},
                        br: {x:-2, y:-1}
                    },
                    terrain: "\nR\nR\nR\n"
                });

                expect(function() { map = map.merge(slice); }).not.toThrow();

                // The bounds must reflect the merge
                var bounds = {
                    tl: {x: -2, y: 1},
                    br: {x:  0, y: -1}
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
                    bounds: {
                        tl: {x:2, y: 1},
                        br: {x:2, y:-1}
                    },
                    terrain: "\nR\nR\nR\n"
                });

                expect(function() { map = map.merge(slice); }).not.toThrow();

                var bounds = {
                    tl: {x: 0, y: 1},
                    br: {x: 2, y: -1}
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
