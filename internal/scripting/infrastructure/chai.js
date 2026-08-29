// A small chai-like assertion API for pm.expect(...), and the pm.test(...)
// result collector. Pure JS on purpose: goja already runs full ES5.1+, so
// there's no need for Go glue beyond exposing the two entry points below.

var __pmTestResults = [];

function pmTest(name, fn) {
  try {
    fn();
    __pmTestResults.push({ name: name, passed: true, error: "" });
  } catch (e) {
    __pmTestResults.push({ name: name, passed: false, error: (e && e.message) ? e.message : String(e) });
  }
}

function __pmMakeExpectation(actual, negate) {
  function describe(v) {
    try {
      return JSON.stringify(v);
    } catch (e) {
      return String(v);
    }
  }
  function assert(cond, message) {
    if (negate ? cond : !cond) {
      throw new Error("expected " + describe(actual) + (negate ? " not " : " ") + message);
    }
  }

  var exp = {};
  Object.defineProperty(exp, "not", { get: function () { return __pmMakeExpectation(actual, !negate); } });
  Object.defineProperty(exp, "to", { get: function () { return exp; } });
  Object.defineProperty(exp, "be", { get: function () { return exp; } });
  Object.defineProperty(exp, "have", { get: function () { return exp; } });
  Object.defineProperty(exp, "been", { get: function () { return exp; } });

  Object.defineProperty(exp, "true", { get: function () { assert(actual === true, "to be true"); return exp; } });
  Object.defineProperty(exp, "false", { get: function () { assert(actual === false, "to be false"); return exp; } });
  Object.defineProperty(exp, "ok", { get: function () { assert(!!actual, "to be truthy"); return exp; } });
  Object.defineProperty(exp, "exist", {
    get: function () { assert(actual !== null && actual !== undefined, "to exist"); return exp; },
  });

  exp.equal = function (expected) {
    assert(actual === expected, "to equal " + describe(expected));
    return exp;
  };
  exp.eql = function (expected) {
    assert(describe(actual) === describe(expected), "to deep-equal " + describe(expected));
    return exp;
  };
  exp.above = function (n) {
    assert(actual > n, "to be above " + n);
    return exp;
  };
  exp.below = function (n) {
    assert(actual < n, "to be below " + n);
    return exp;
  };
  exp.include = function (item) {
    var cond;
    if (typeof actual === "string" || Array.isArray(actual)) {
      cond = actual.indexOf(item) !== -1;
    } else if (actual && typeof actual === "object") {
      cond = Object.prototype.hasOwnProperty.call(actual, item);
    } else {
      cond = false;
    }
    assert(cond, "to include " + describe(item));
    return exp;
  };

  return exp;
}

function pmExpect(actual) {
  return __pmMakeExpectation(actual, false);
}
