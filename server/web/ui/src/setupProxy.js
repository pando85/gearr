const { createProxyMiddleware } = require('http-proxy-middleware');

module.exports = function (app) {
    app.use(
        '/api',
        createProxyMiddleware({
            target: 'http://localhost:8080',
            // I don't understand the f**** http-proxy-middleware. This seems needed.
            // Also, it starts a f***** client trying to connect to `/ws` ¯\_(ツ)_/¯
            ws: true,
        }),
    );
    app.use(
        '/ws/job',
        createProxyMiddleware({
            target: 'ws://localhost:8080',
            ws: true,
            logLevel: 'debug',
        }),
    );
};
