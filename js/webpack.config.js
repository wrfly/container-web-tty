const UglifyJSPlugin = require('uglifyjs-webpack-plugin');

module.exports = {
    entry: "./src/main.ts",
    mode: "production",
    output: {
        filename: "gotty-bundle.js"
    },
    devtool: "source-map",
    resolve: {
        extensions: [".ts", ".tsx", ".js"],
    },
    module: {
        rules: [{
                test: /\.tsx?$/,
                loader: "ts-loader",
                exclude: /node_modules/
            },
            {
                test: /\.js$/,
                include: /node_modules/,
                loader: 'license-loader'
            },
            {
                test: /\.js$/,
                include: /src/,
                use: {
                    loader: 'babel-loader',
                    query: {
                        presets: ["es2015"]
                    }
                }
            }
        ]
    },
    plugins: [
        new UglifyJSPlugin()
    ]
};