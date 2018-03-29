# Entrepot

Artifact repository.

## Usage

It's possible to use this repository with your Maven or SBT build (as resolver).

### Maven 2/3

Following code can be merged into your `HOME/.m2/settings.xml` to be able to use this repository:

```xml
<settings>
  <profiles>
    <profile>
      <!-- ... -->
      <repositories>
        <repository>
          <id>entrepot-releases</id>
          <name>Entrepot Maven2 Repository (releases)</name>
          <url>https://raw.github.com/zengularity/entrepot/master/releases</url>
          <releases>
            <enabled>true</enabled>
          </releases>
          <snapshots>
            <enabled>true</enabled>
          </snapshots>
        </repository>
      </repositories>
    </profile>
  </profiles>
</settings>
```

Url can be changed to https://raw.github.com/zengularity/entrepot/snapshots to use snapshots artifacts.

### SBT

Add this repo to your SBT config:

```scala
resolvers ++= Seq(
  "Entrepot Releases" at "https://raw.github.com/zengularity/entrepot/master/releases",
  "Entrepot Snapshots" at "https://raw.github.com/zengularity/entrepot/master/snapshots")

```

## Publish

This section is about how to publish a project to this repository so it can be shared.

### SBT

In the `build.sbt`, settings as bellow can be configured.

```scala
publishTo in ThisBuild := Some {
  import Resolver.mavenStylePatterns

  val repoDir = sys.env.get("REPO_PATH").map { path =>
    new java.io.File(path)
  }.getOrElse(sys.error("REPO_PATH is not set"))

  Resolver.file("repo", repoDir)
}
```

Assuming that the `entrepot` repository is cloned locally at `/path/to/entrepot`, then the project can be published using SBT with the following steps.

- Set the environment variable `REPO_PATH`; e.g. `export REPO_PATH=/path/to/entrepot/snapshots/`
- Run the command `sbt publish` .

```sh
export REPO_PATH=/path/to/entrepot/snapshots/
# OR: export REPO_PATH=/path/to/entrepot/releases/

# in project directory with the configured build.sbt
sbt publish

# Push the publication
cd /path/to/entrepot/
git add snapshots/
git commit -m "Publish X"
git push
```
