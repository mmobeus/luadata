use std::io::{self, Read};
use std::process;

use clap::{Parser, Subcommand};

use luadata::options::{
    ArrayMode, EmptyTableMode, ParseConfig, StringTransform, StringTransformMode,
};

#[derive(Parser)]
#[command(name = "luadata")]
#[command(about = "Convert Lua data files to JSON")]
struct Cli {
    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand)]
enum Commands {
    /// Convert a Lua data file to JSON (use - for stdin)
    Tojson {
        /// Input file path (use - for stdin)
        file: String,

        #[command(flatten)]
        opts: SharedOpts,
    },
    /// Check that a Lua data file parses successfully
    Validate {
        /// Input file path
        file: String,

        #[command(flatten)]
        opts: SharedOpts,
    },
}

#[derive(Parser)]
struct SharedOpts {
    /// How to render empty tables: null, omit, array, object
    #[arg(long = "empty-table", default_value = "null")]
    empty_table: String,

    /// Array detection mode: none, index-only, sparse
    #[arg(long = "array-mode", default_value = "sparse")]
    array_mode: String,

    /// Max gap between keys for sparse array detection
    #[arg(long = "array-max-gap", default_value = "20")]
    array_max_gap: usize,

    /// Max string length before transform (0 = disabled)
    #[arg(long = "string-max-len", default_value = "0")]
    string_max_len: usize,

    /// String transform mode: truncate, empty, redact, replace
    #[arg(long = "string-mode", default_value = "truncate")]
    string_mode: String,

    /// Replacement text for replace mode
    #[arg(long = "string-replacement", default_value = "")]
    string_replacement: String,
}

fn build_config(opts: &SharedOpts) -> Result<ParseConfig, String> {
    let mut config = ParseConfig::new();

    config.empty_table_mode = match opts.empty_table.as_str() {
        "null" => EmptyTableMode::Null,
        "omit" => EmptyTableMode::Omit,
        "array" => EmptyTableMode::Array,
        "object" => EmptyTableMode::Object,
        v => {
            return Err(format!(
                "unknown --empty-table value: {:?} (valid: null, omit, array, object)",
                v
            ));
        }
    };

    config.array_mode = Some(match opts.array_mode.as_str() {
        "none" => ArrayMode::None,
        "index-only" => ArrayMode::IndexOnly,
        "sparse" => ArrayMode::Sparse {
            max_gap: opts.array_max_gap,
        },
        v => {
            return Err(format!(
                "unknown --array-mode value: {:?} (valid: none, index-only, sparse)",
                v
            ));
        }
    });

    if opts.string_max_len > 0 {
        let mode = match opts.string_mode.as_str() {
            "truncate" => StringTransformMode::Truncate,
            "empty" => StringTransformMode::Empty,
            "redact" => StringTransformMode::Redact,
            "replace" => StringTransformMode::Replace,
            v => {
                return Err(format!(
                    "unknown --string-mode value: {:?} (valid: truncate, empty, redact, replace)",
                    v
                ));
            }
        };

        config.string_transform = Some(StringTransform {
            max_len: opts.string_max_len,
            mode,
            replacement: opts.string_replacement.clone(),
        });
    }

    Ok(config)
}

fn main() {
    let cli = Cli::parse();

    match cli.command {
        Commands::Tojson { file, opts } => {
            let config = match build_config(&opts) {
                Ok(c) => c,
                Err(e) => {
                    eprintln!("Error: {}", e);
                    process::exit(1);
                }
            };

            let result = if file == "-" {
                let mut input = String::new();
                if let Err(e) = io::stdin().read_to_string(&mut input) {
                    eprintln!("Error reading stdin: {}", e);
                    process::exit(1);
                }
                luadata::text_to_json("stdin", &input, config)
            } else {
                luadata::file_to_json(&file, config)
            };

            match result {
                Ok(json) => {
                    println!("{}", json);
                }
                Err(e) => {
                    eprintln!("Error converting: {}", e);
                    process::exit(1);
                }
            }
        }
        Commands::Validate { file, opts } => {
            let config = match build_config(&opts) {
                Ok(c) => c,
                Err(e) => {
                    eprintln!("Error: {}", e);
                    process::exit(1);
                }
            };

            if let Err(e) = luadata::file_to_json(&file, config) {
                eprintln!("Error validating {}: {}", file, e);
                process::exit(1);
            }
        }
    }
}
