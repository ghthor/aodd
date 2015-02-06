define(["client/updateBuffer",
       "underscore",
       "lib/jasmine"
], function(UpdateBuffer, _) {
    describe("The buffer that merges multiple updates into a single update", function() {
        var buffer;
        var entity0 = {id: 0};
        var entity1 = {id: 1};
        var terrainMap;

        describe("merging", function() {
            beforeEach(function() {
                terrainMap = {
                    bounds: {
                        tl: {x:-1, y: 1},
                        br: {x: 1, y:-1}
                    },
                    terrain: "\nDGG\nGDG\nGGD\n"
                };

                buffer = new UpdateBuffer();
                buffer.merge({
                    time: 1,
                    entities:[entity0, entity1],
                    removed: null,
                    terrainMap: terrainMap
                });

                this.addMatchers({
                    toOnlyContain: function(expected) {
                        if (this.actual.length === 0 && expected.length > 0) {
                            this.message = function() {
                                return "Expected to contain " + expected + "but was empty";
                            };
                            return false;
                        }

                        var diff = _.difference(this.actual, expected);
                        this.message = function() {
                            return "Expected to contain " + expected + " but was missing " + diff;
                        };
                        return diff.length === 0;
                    }
                });
            });

            it("a single update", function() {
                var update = buffer.merged();
                expect(update.time).toBe(1);
                expect(update.entities).toOnlyContain([entity0, entity1]);
                expect(update.removed.length).toBe(0);
                expect(update.terrainMap).toEqual(terrainMap);
            });

            it("a blank update is an error", function() {
                expect(function() {
                    buffer.merge({
                        time: 2,
                        entities: null,
                        removed: null
                    });
                }).toThrow("blank update");
            });

            it("an update that was in the past is an error", function() {
                expect(function() {
                    buffer.merge({
                        time: 0,
                        entities: [entity0],
                        removed: null
                    });
                }).toThrow("update was in the past");
            });

            it("will update the time", function() {
                for (var i = buffer.merged().time + 1; i <= 10; i++) {
                    buffer.merge({
                        time: i,
                        entities: [entity0],
                        removed: null
                    });
                    expect(buffer.merged().time).toBe(i);
                }
            });

            it("an update that adds an entity", function() {
                var entity2 = { id: 2 };
                buffer.merge({
                    time: 2,
                    entities:[entity2],
                    removed: null
                });

                var update = buffer.merged();
                expect(update.entities).toOnlyContain([entity0, entity1, entity2]);
            });

            it("an update that contains an updated entity", function() {
                var entity0modified = { id: 0 };
                buffer.merge({
                    time: 2,
                    entities: [entity0modified],
                    removed: null
                });

                var update = buffer.merged();
                expect(update.entities).toOnlyContain([entity0modified, entity1]);
            });

            it("an update that removes an entity that exists in the buffer", function() {
                buffer.merge({
                    time: 2,
                    entities: null,
                    removed: [entity0]
                });

                var update = buffer.merged();
                expect(update.entities).toOnlyContain([entity1]);
                expect(update.removed.length).toBe(0);
            });

            it("an update that removes an entity that doesn't exist in the buffer", function() {
                var entity2 = { id: 2 };
                buffer.merge({
                    time: 2,
                    entities: null,
                    removed: [entity2]
                });

                var update = buffer.merged();
                expect(update.entities).toOnlyContain([entity0, entity1]);
                expect(update.removed).toOnlyContain([entity2]);

                var entity3 = { id: 3 };
                buffer.merge({
                    time: 3,
                    entities: null,
                    removed: [entity3]
                });

                update = buffer.merged();
                expect(update.entities).toOnlyContain([entity0, entity1]);
                expect(update.removed).toOnlyContain([entity2, entity3]);

                var entity2modified = { id: 2 };
                buffer.merge({
                    time: 4,
                    entities: null,
                    removed: [entity2modified]
                });

                update = buffer.merged();
                expect(update.entities).toOnlyContain([entity0, entity1]);
                expect(update.removed).toOnlyContain([entity2modified, entity3]);
            });

            describe("an update that contains a terrain slice", function() {
                it("to the north", function() {
                    var slice = {
                        bounds: {
                            tl: {x: -1, y: 2},
                            br: {x:  1, y: 2}
                        },
                        terrain: "\nRRR\n"
                    };

                    expect(function() {
                        buffer.merge({
                            time: 2,
                            entities: null,
                            removed: null,
                            terrainMap: slice
                        });
                    }).not.toThrow();

                    //"\nDGG\nGDG\nGGD\n"
                    expect(buffer.merged().terrainMap).toEqual({
                        bounds: {
                            tl: {x: -1, y: 2},
                            br: {x:  1, y: 0}
                        },
                        terrain: "\nRRR\nDGG\nGDG\n"
                    });
                });

                it("to the south", function() {
                    var slice = {
                        bounds: {
                            tl: {x: -1, y:-2},
                            br: {x:  1, y:-2}
                        },
                        terrain: "\nRRR\n"
                    };

                    expect(function() {
                        buffer.merge({
                            time: 2,
                            entities: null,
                            removed: null,
                            terrainMap: slice
                        });
                    }).not.toThrow();

                    expect(buffer.merged().terrainMap).toEqual({
                        bounds: {
                            tl: {x: -1, y:  0},
                            br: {x:  1, y: -2}
                        },
                        terrain: "\nGDG\nGGD\nRRR\n"
                    });
                });

                it("to the west", function() {
                    var slice = {
                        bounds: {
                            tl: {x: -2, y: 1},
                            br: {x: -2, y:-1}
                        },
                        terrain: "\nR\nR\nR\n"
                    };

                    expect(function() {
                        buffer.merge({
                            time: 2,
                            entities: null,
                            removed: null,
                            terrainMap: slice
                        });
                    }).not.toThrow();

                    expect(buffer.merged().terrainMap).toEqual({
                        bounds: {
                            tl: {x: -2, y:  1},
                            br: {x:  0, y: -1}
                        },
                        terrain: "\nRDG\nRGD\nRGG\n"
                    });
                });

                it("to the east", function() {
                    var slice = {
                        bounds: {
                            tl: {x: 2, y: 1},
                            br: {x: 2, y:-1}
                        },
                        terrain: "\nR\nR\nR\n"
                    };

                    expect(function() {
                        buffer.merge({
                            time: 2,
                            entities: null,
                            removed: null,
                            terrainMap: slice
                        });
                    }).not.toThrow();

                    expect(buffer.merged().terrainMap).toEqual({
                        bounds: {
                            tl: {x: 0, y:  1},
                            br: {x: 2, y: -1}
                        },
                        terrain: "\nGGR\nDGR\nGDR\n"
                    });
                });
            });
        });
    });
});
