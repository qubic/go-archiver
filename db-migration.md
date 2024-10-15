# Process of migrating an Archiver instance:

1. Stop archiver.
2. Run the [archiver-db-migrator](https://github.com/qubic/archiver-db-migrator) tool, specifying the path of the old database, and the path of where to store the new database directory. Wait for the process to stop.
3. Rename the old database directory to something else, and the new directory to the old name, thus we don't need to modify the archiver configuration.
4. Update the archiver version in the docker compose file.
5. Start archiver.