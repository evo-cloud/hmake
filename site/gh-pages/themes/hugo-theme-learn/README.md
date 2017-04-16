# Hugo Learn Theme

This repository contains a theme for [Hugo](https://gohugo.io/), based on great [Grav Learn Theme](http://learn.getgrav.org/).

Visit the [theme documentation](https://matcornic.github.io/hugo-learn-doc/basics/what-is-this-hugo-theme/) to see what is going on. It is actually built with this theme.

## Installation
Navigate to your themes folder in your Hugo site and use the following commands:
```
$ cd themes
$ git clone https://github.com/matcornic/hugo-theme-learn.git
```

Check that your Hugo version is minimum `0.19` with `hugo version`.

![Overview](https://github.com/matcornic/hugo-theme-learn/raw/master/images/tn.png)

## Usage

- [Visit the documentation](https://matcornic.github.io/hugo-learn-doc/basics/what-is-this-hugo-theme/)

# Main functionalities

- Handle two levels of documentation
- Tip/Note/Info and Warning boxes
- Resize images
- Preview of original image size
- Add shadow or border on images
- Automatic table of contents
- Create buttons (typically used to provide a link to a demo)
- Search using `lunr` index

# TODO

- [Handling more than 2 levels in documentation](https://github.com/matcornic/hugo-theme-learn/issues/11)
- [Handling videos](https://github.com/matcornic/hugo-theme-learn/issues/13)
- [Add optional button to create doc issue (like github)](https://github.com/matcornic/hugo-theme-learn/issues/14)

# Troubleshooting

Changes have been made for automatically creating previous and next arrows. These changes restructured the theme and you might have compatibility problems when updating to the latest version of the theme. Please read [PR#36](https://github.com/matcornic/hugo-theme-learn/pull/36) to update your documentation with latest guidelines.

If you don't want to update your documentation, use the git tag `1.0.0` 

```shell
cd themes/hugo-theme-learn
git checkout tags/1.0.0
```
