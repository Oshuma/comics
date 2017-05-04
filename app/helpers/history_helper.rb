module HistoryHelper

  def link_to_history_group(history)
    group = current_user.groups.find_by(name: history.group_name)
    if group.present?
      link_to(group.name, group)
    else
      link_to(history.group_name, new_comic_path(group_name: history.group_name))
    end
  end

  def link_to_history_comic(history)
    comic = current_user.comics.find_by(filename: history.comic_name)
    if comic.present?
      link_to(comic.filename, comic)
    else
      history.comic_name
    end
  end

end
