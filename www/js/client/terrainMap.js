define(["client/sprite/terrain",
       "underscore"
], function(Sprite, _) {
    var parseTerrainMap = function(terrainStr) {
        var rows = terrainStr.trim("\n").split("\n");
        _.each(rows, function(row, y) {
            rows[y] = row.split("");
        });
        return rows;
    };

    var TerrainMap = function(def, canvas, grid) {
        var map = this;

        var bounds = map.bounds = def.bounds,
            h = bounds.tl.y - bounds.br.y + 1,
            w = bounds.br.x - bounds.tl.x + 1;

        var terrain = parseTerrainMap(def.terrain);

        // Sanity checks
        if (h !== terrain.length) {
            throw "invalid terrain height";
        }

        for (var i = 0; i < terrain.length; i++) {
            if (w !== terrain[i].length) {
                throw "invalid terrain width";
            }
        }

        // This object MUST be mocked in specs
        var sprite = map.sprite = TerrainMap.sprite;

        // This will have to be smarter in the future to account
        // for the neighboring terrain types
        var sprites = map.sprites = _.map(terrain, function(row, y) {
            return _.map(row, function(type, x) {
                // Get the coordinates
                var tl = bounds.tl;
                var cell = {
                    x: x + tl.x,
                    y: -y + tl.y
                };

                // Create tile actor
                var tile;
                switch (type) {
                case "D":
                    tile = sprite.makeDirtTile(cell);
                    break;
                case "R":
                    tile = sprite.makeRockTile(cell);
                    break;
                default:
                    tile = sprite.makeGrassTile(cell);
                    break;
                }
                return tile;
            });
        });

        if (!map.isSlice()) {
            map.center = function() {
                var y = (sprites.length - 1) / 2,
                x = (sprites[0].length - 1) / 2;

                var tile = sprites[y][x];

                return tile.cell;
            };

            canvas.width  = w * grid;
            canvas.height = h * grid;
            var ctx = canvas.getContext("2d");
            _.each(sprites, function(row, y) {
                _.each(row, function(tile, x) {
                    tile.paint(ctx, x * grid, y * grid);
                });
            });

            var mergeWest = function(slice, map) {
                // Canvas shifts east
                ctx.drawImage(canvas, 0, 0, (w-1)*grid, h*grid, grid, 0, (w-1)*grid, h*grid);
                _.each(map.sprites, function(row, y) {
                    row.pop();

                    var tile = slice.sprites[y][0];
                    tile.paint(ctx, 0, y * grid);
                    row.unshift(tile);
                });
            };

            var mergeEast = function(slice, map) {
                // Canvas shifts west
                ctx.drawImage(canvas, grid, 0, (w-1)*grid, h*grid, 0, 0, (w-1)*grid, h*grid);
                _.each(map.sprites, function(row, y) {
                    row.shift();

                    var tile = slice.sprites[y][0];
                    tile.paint(ctx, (w - 1) * grid, y * grid);
                    row.push(tile);
                });
            };

            var mergeSouth = function(slice, map) {
                // Canvas shifts north
                ctx.drawImage(canvas, 0, grid, w*grid, (h-1)*grid, 0, 0, w*grid, (h-1)*grid);
                map.sprites.shift();

                var row = slice.sprites[0];
                _.each(row, function(tile, x) {
                    tile.paint(ctx, x * grid, (h - 1) * grid);
                });
                map.sprites.push(row);
            };

            var mergeNorth = function(slice, map) {
                // Canvas shifts south
                ctx.drawImage(canvas, 0, 0, w*grid, (h-1)*grid, 0, grid, w*grid, (h-1)*grid);
                map.sprites.pop();

                var row = slice.sprites[0];
                _.each(row, function(tile, x) {
                    tile.paint(ctx, x*grid, 0);
                });
                map.sprites.unshift(row);
            };

            var mergeVerticalSlice   = makeVerticalMerge(mergeWest, mergeEast);
            var mergeHorizontalSlice = makeHorizontalMerge(mergeSouth, mergeNorth);
            map.merge = function(slice) {
                return merge(slice, this, mergeVerticalSlice, mergeHorizontalSlice);
            };

        }
    };

    // Module initialization
    TerrainMap.initialize = function(director) {
        if (_.isUndefined(TerrainMap.sprite)) {
            TerrainMap.sprite = new Sprite(Sprite.makeImage(director));
        } else if (TerrainMap.sprite.prototype !== Sprite.prototype) {
            TerrainMap.sprite = new Sprite(Sprite.makeImage(director));
        }
    };

    var makeVerticalMerge = function(mergeWest, mergeEast) {
        return function(slice, map) {
            var mtl = map.bounds.tl,
                mbr = map.bounds.br,
                stl = slice.bounds.tl,
                sbr = slice.bounds.br;

            if (stl.x < mtl.x) {
                // shift west
                mtl.x = stl.x;
                mbr.x -= 1;

                mergeWest(slice, map);

            } else if (sbr.x > mbr.x) {
                // merge east
                mbr.x = sbr.x;
                mtl.x += 1;

                mergeEast(slice, map);
            } else {
                throw "invalid terrain map horizontal merge";
            }
            return map;
        };
    };

    var makeHorizontalMerge = function(mergeSouth, mergeNorth) {
        return function(slice, map) {
            var mtl = map.bounds.tl,
                mbr = map.bounds.br,
                stl = slice.bounds.tl,
                sbr = slice.bounds.br;

            if (stl.y > mtl.y) {
                // merge south
                mtl.y = stl.y;
                mbr.y += 1;

                mergeNorth(slice, map);
            } else if (sbr.y < mbr.y) {
                // merge north
                mbr.y = sbr.y;
                mtl.y -= 1;

                mergeSouth(slice, map);
            } else {
                throw "invalid terrain map vertical merge";
            }
            return map;
        };
    };

    var merge = function(slice, map, mergeVerticalSlice, mergeHorizontalSlice) {
        var tl = slice.bounds.tl,
            br = slice.bounds.br;

        if (tl.x === br.x) {
            map = mergeVerticalSlice(slice, map);
        } else if (tl.y === br.y) {
            map = mergeHorizontalSlice(slice, map);
        } else {
            throw "invalid terrain map merge";
        }
        return map;
    };

    TerrainMap.prototype = {
        isSlice: function() {
            var tl = this.bounds.tl,
            br = this.bounds.br;

            return tl.x === br.x || tl.y === br.y;
        }
    };

    // Module static exports
    TerrainMap.parseTerrain        = parseTerrainMap;
    TerrainMap.makeHorizontalMerge = makeHorizontalMerge;
    TerrainMap.makeVerticalMerge   = makeVerticalMerge;
    TerrainMap.merge = merge;

    return TerrainMap;
});
