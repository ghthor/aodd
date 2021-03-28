define(["app",
       "github.com/ghthor/filu/rpg2d/coord",
       "ui/canvas/sprite/terrain",
       "underscore",
], function(app, coord, Sprite, _) {
    var parseTerrainMap = function(terrainStr) {
        var rows = terrainStr.trim("\n").split("\n");
        _.each(rows, function(row, y) {
            rows[y] = row.split("");
        });
        return rows;
    };

    // Constructs a new terrain map.
    //
    // terrainMap worldterrain.MapState
    //            http://godoc.org/github.com/ghthor/filu/rpg2d/worldterrain#MapState
    //
    // canvas     canvas dom element
    // tileSz     int
    var TerrainMap = function(terrainMap, canvas, tileSz, client) {
        var map = this;

        console.log(terrainMap);
        var bounds = map.Bounds = terrainMap.Bounds;
        var h = bounds.TopL.Y - bounds.BotR.Y + 1,
            w = bounds.BotR.X - bounds.TopL.X + 1;

        var terrain = parseTerrainMap(terrainMap.Terrain);

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
        var sprite = TerrainMap.sprite;

        // Construct a tile from a type and a cell
        var newTile = function(type, cell) {
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
        };

        // This will have to be smarter in the future to account
        // for the neighboring terrain types
        var drawTile = function(type, cell) {
            var tile = newTile(type, cell);

            var x = cell.X - bounds.TopL.X,
                y = -(cell.Y - bounds.TopL.Y);

            tile.paint(ctx, x * tileSz, y * tileSz);
        };

        map.center = function() {
            var x = (w - 1) / 2,
                y = (h - 1) / 2;

            return {
                X: bounds.TopL.X + x,
                Y: bounds.TopL.Y - y,
            };
        };

        canvas.width  = w * tileSz;
        canvas.height = h * tileSz;
        var ctx = canvas.getContext("2d");

        var drawTerrain = function(terrain) {
            _.each(terrain, function(row, y) {
                _.each(row, function(type, x) {
                    drawTile(type, {
                        X: x + bounds.TopL.X,
                        Y: -y + bounds.TopL.Y
                    });
                });
            });
        };

        drawTerrain(terrain);

        // Viewport moved to the north
        var shiftSouth = function(mag) {
            bounds.TopL.Y += mag;
            bounds.BotR.Y += mag;

            ctx.drawImage(canvas,
                0, 0, w*tileSz, (h-mag)*tileSz,
                0, tileSz*mag, w*tileSz, (h-mag)*tileSz);
        };

        // Viewport moved to the east
        var shiftWest = function(mag) {
            bounds.TopL.X += mag;
            bounds.BotR.X += mag;

            ctx.drawImage(canvas,
                tileSz*mag, 0, (w-mag)*tileSz, h*tileSz,
                0, 0, (w-mag)*tileSz, h*tileSz);
        };

        // Viewport moved to the south
        var shiftNorth = function(mag) {
            bounds.TopL.Y -= mag;
            bounds.BotR.Y -= mag;

            ctx.drawImage(canvas,
                0, tileSz*mag, w*tileSz, (h-mag)*tileSz,
                0, 0, w*tileSz, (h-mag)*tileSz);
        };

        // Viewport moved to the west
        var shiftEast = function(mag) {
            bounds.TopL.X -= mag;
            bounds.BotR.X -= mag;

            ctx.drawImage(canvas,
                0, 0, (w-mag)*tileSz, h*tileSz,
                tileSz*mag, 0, (w-mag)*tileSz, h*tileSz);
        };

        client.on(app.EV_TERRAIN_RESET, function(map) {
            bounds = map.bounds;

            var terrain = parseTerrainMap(terrainMap.Terrain);
            drawTerrain(terrain);

        });

        client.on(app.EV_TERRAIN_CANVAS_SHIFT, function(dir, mag) {
            console.log("shift", dir, mag);
            switch (dir) {
            case coord.North:
                shiftNorth(mag);
                break;
            case coord.East:
                shiftEast(mag);
                break;
            case coord.South:
                shiftSouth(mag);
                break;
            case coord.West:
                shiftWest(mag);
                break;
            default:
                console.log("unknown canvas shift direction", dir);
            }
        });

        client.on(app.EV_TERRAIN_DRAW_TILE, function(ttype, cell) {
            drawTile(ttype, cell);
        });
    };

    // Initializes the module with a director
    // which is the source of the tileset sprite.
    TerrainMap.initialize = function(director) {
        if (_.isUndefined(TerrainMap.sprite)) {
            TerrainMap.sprite = new Sprite(Sprite.makeImage(director));
        } else if (TerrainMap.sprite.prototype !== Sprite.prototype) {
            TerrainMap.sprite = new Sprite(Sprite.makeImage(director));
        }
    };

    // Module static exports
    TerrainMap.parseTerrain = parseTerrainMap;

    return TerrainMap;
});
