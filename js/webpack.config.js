const UglifyJSPlugin = require('uglifyjs-webpack-plugin');

module.exports = {
    entry: "./src/main.ts",
    mode: "production",
    output: {
        filename: "gotty-bundle.js"
    },
    mode: "production",
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
            }
        ]
    },
    plugins: [
        new UglifyJSPlugin()
    ]
};