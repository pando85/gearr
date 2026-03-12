const fs = require('fs');
const glob = require('glob');
const paths = glob.sync('./node_modules/**/schema-utils/dist/validate.js');

paths.forEach(path => {
  if (fs.existsSync(path)) {
    let content = fs.readFileSync(path, 'utf8');
    if (!content.includes('ajv-formats')) {
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
      console.log('Patched: ' + path);
    }
  }
});