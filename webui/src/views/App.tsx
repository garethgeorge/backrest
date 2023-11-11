import test from "node:test";
import * as React from "react";
import _ from "lodash";

export const App = () => {
  const [searchTerm, setSearchTerm] = React.useState("");
  const [results, setResults] = React.useState<any>();

  return (
    <div>
      <h1>Guidebook FS Search</h1>
      <div>
        Search:
        <input
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
        ></input>
      </div>
      <pre>
        <code>{"RESULTS: " + JSON.stringify(results, null, 2)}</code>
      </pre>
    </div>
  );
};
