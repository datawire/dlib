# Developing dlib

## To do a dlib release

1. Create a branch.

2. Fill in the release date in `NEWS.md`, and commit that.

3. Create an annotated `vSEMVER` git tag:

   ```shell
   git tag --annotate --message='Release 1.2.0' v1.2.0
   ```

   Having looked at the release notes in `NEWS.md`, you should have an
   idea of whether this needs to be a minor version bump or a patch
   version bump.

4. Push the tag and branch to GitHub:

   ```shell
   git push origin v1.2.0 my-branch-name
   ```

5. Create a pull-request for the branch.

That's it!
