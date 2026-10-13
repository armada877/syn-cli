# Omarchy setup spec

OmarchyPath uses ~/.config/omarchy/themed/syn.toml.tpl under the supplied home path.
InstallOmarchy receives an explicit target path.
It creates missing parent directories and writes the embedded template with mode 0644.
An identical existing regular file makes installation succeed without another write.
A different existing file produces an error and remains unchanged.
A symlink target produces an error and remains intact.
Directory creation, write, and close failures produce errors.
Setup changes no other Omarchy files and runs no Omarchy commands.
The template contains only the ten supported scalar keys.
