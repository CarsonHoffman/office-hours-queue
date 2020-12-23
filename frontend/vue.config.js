const fs = require('fs');

module.exports = {
	pages: {
		index: {
			entry: 'src/main.ts',
			title: 'EECS Office Hours',
		},
	},
	configureWebpack: (config) => {
		if (process.env.NODE_ENV !== 'production') {
			config.devServer = {
				proxy: {
					'^/api': {
						target: 'https://lvh.me:8080',
						ws: true,
						// changeOrigin didn't work for WebSocket connections.
						onProxyReqWs: function(request) {
							request.setHeader('origin', 'https://lvh.me:8080');
						},
					},
				},
				https: true,
				key: fs.readFileSync('../deploy/secrets/certs/localhost-key.pem'),
				cert: fs.readFileSync('../deploy/secrets/certs/localhost.pem'),
				transportMode: 'ws',
			};
		}
	},
};
