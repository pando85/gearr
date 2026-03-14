const fs = require('fs');
const glob = require('glob');
const paths = glob.sync('./node_modules/**/schema-utils/dist/validate.js');

paths.forEach(path => {
  if (fs.existsSync(path)) {
    const pkgPath = path.replace('/dist/validate.js', '/package.json');
    let pkgVersion = 'unknown';
    try {
      const pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf8'));
      pkgVersion = pkg.version;
    } catch (e) {}
    
    let content = fs.readFileSync(path, 'utf8');
    const isV4 = pkgVersion.startsWith('4.');
    
    if (isV4 && !content.includes('ajv-formats')) {
      content = content.replace(
        'var _ajvKeywords = _interopRequireDefault(require("ajv-keywords"));',
        'var _ajvKeywords = _interopRequireDefault(require("ajv-keywords"));\nvar _ajvFormats = _interopRequireDefault(require("ajv-formats"));'
      );
      content = content.replace(
        "const ajvKeywords = require('ajv-keywords');",
        "const ajvKeywords = require('ajv-keywords');\nconst ajvFormats = require('ajv-formats');"
      );
      content = content.replace(
        "(0, _ajvKeywords.default)(ajv, ['instanceof', 'formatMinimum', 'formatMaximum', 'patternRequired'])",
        "(0, _ajvFormats.default)(ajv);\n(0, _ajvKeywords.default)(ajv, ['instanceof', 'patternRequired'])"
      );
      content = content.replace(
        "ajvKeywords(ajv, ['instanceof', 'formatMinimum', 'formatMaximum', 'patternRequired'])",
        "ajvFormats(ajv);\najvKeywords(ajv, ['instanceof', 'patternRequired'])"
      );
      fs.writeFileSync(path, content);
      console.log('Patched: ' + path + ' (v' + pkgVersion + ')');
    } else if (!isV4) {
      console.log('Skipped: ' + path + ' (v' + pkgVersion + ' - uses ajv@6)');
    }
  }
});

const babelLoaderPaths = glob.sync('./node_modules/**/babel-loader/lib/index.js');

babelLoaderPaths.forEach(path => {
  if (fs.existsSync(path)) {
    let content = fs.readFileSync(path, 'utf8');
    
    if (content.includes('const validateOptions = require("schema-utils");')) {
      content = content.replace(
        'const validateOptions = require("schema-utils");',
        'const { validate: validateOptions } = require("schema-utils");'
      );
      fs.writeFileSync(path, content);
      console.log('Patched babel-loader: ' + path);
    }
  }
});